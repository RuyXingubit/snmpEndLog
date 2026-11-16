"""Syslog receiver — UDP and TCP servers for receiving syslog messages.

Security features:
- IP allowlist: only accepts syslog from IPs registered as devices in the database
- Message size limit: 8192 bytes per message (RFC 5424)
- Buffer cap: 50,000 messages maximum to prevent memory exhaustion
- Silent drop of unauthorized IPs (no logging to prevent log flooding)
"""

import asyncio
import ipaddress
import logging
from typing import Any

import db
from logs.parser import apply_registered_device_identity, parse_syslog_message

logger = logging.getLogger(__name__)

# Security constants
MAX_MESSAGE_SIZE = 8192  # bytes, per RFC 5424
MAX_BUFFER_SIZE = 50_000  # maximum buffered messages before dropping
ALLOWLIST_REFRESH_INTERVAL = 30  # seconds


def normalize_ip(ip_str: str) -> str:
    """Normalize IPv4/IPv6 (including IPv4-mapped IPv6) to canonical string form."""
    try:
        addr = ipaddress.ip_address(ip_str)
        if isinstance(addr, ipaddress.IPv6Address) and addr.ipv4_mapped:
            return str(addr.ipv4_mapped)
        return str(addr)
    except ValueError:
        return ip_str


def _resolve_allowed_device(
    allowed_ips: dict[str, Any], source_ip: str
) -> tuple[int, str, str] | None:
    """Resolve an allowed source IP to (device_id, hostname, sys_name), or None."""
    entry = allowed_ips.get(source_ip)
    if entry is None:
        return None
    if isinstance(entry, int):
        return (entry, "", "")
    if isinstance(entry, dict) and entry.get("id") is not None:
        return (
            int(entry["id"]),
            str(entry.get("hostname") or ""),
            str(entry.get("sys_name") or ""),
        )
    return None


class SyslogUDPProtocol(asyncio.DatagramProtocol):
    """UDP syslog receiver (RFC 3164/5424 over UDP)."""

    def __init__(
        self,
        buffer: list[dict[str, Any]],
        buffer_lock: asyncio.Lock,
        allowed_ips: dict[str, Any],
    ):
        self._buffer = buffer
        self._buffer_lock = buffer_lock
        self._allowed_ips = allowed_ips

    def connection_made(self, transport: asyncio.DatagramTransport) -> None:
        self._transport = transport

    def datagram_received(self, data: bytes, addr: tuple[str, int]) -> None:
        source_ip = normalize_ip(addr[0])

        # Security: O(1) in-memory IP allowlist check — silently drop unknown IPs
        resolved = _resolve_allowed_device(self._allowed_ips, source_ip)
        if resolved is None:
            return
        device_id, dev_hostname, dev_sys_name = resolved

        # Security: reject oversized messages
        if len(data) > MAX_MESSAGE_SIZE:
            return

        # Security: reject if buffer is full
        if len(self._buffer) >= MAX_BUFFER_SIZE:
            return

        try:
            raw = data.decode("utf-8", errors="replace").replace("\x00", "").strip()
            if not raw:
                return
            parsed = parse_syslog_message(raw, source_ip)
            apply_registered_device_identity(
                parsed, device_id, source_ip, dev_hostname, dev_sys_name
            )

            self._buffer.append(parsed)

        except Exception:
            logger.exception("Error processing UDP syslog from %s", source_ip)


async def handle_tcp_client(
    reader: asyncio.StreamReader,
    writer: asyncio.StreamWriter,
    buffer: list[dict[str, Any]],
    buffer_lock: asyncio.Lock,
    allowed_ips: dict[str, Any],
) -> None:
    """Handle a single TCP syslog client connection."""
    addr = writer.get_extra_info("peername")
    source_ip = normalize_ip(addr[0]) if addr else ""

    # Security: immediately close TCP connections from unauthorized IPs
    if _resolve_allowed_device(allowed_ips, source_ip) is None:
        writer.close()
        try:
            await writer.wait_closed()
        except Exception:
            pass
        return

    logger.debug("TCP syslog connection from %s", source_ip)

    try:
        while True:
            data = await asyncio.wait_for(reader.readline(), timeout=300)
            if not data:
                break

            resolved = _resolve_allowed_device(allowed_ips, source_ip)
            if resolved is None:
                break
            device_id, dev_hostname, dev_sys_name = resolved

            # Security: reject oversized messages
            if len(data) > MAX_MESSAGE_SIZE:
                continue

            # Security: reject if buffer is full
            if len(buffer) >= MAX_BUFFER_SIZE:
                continue

            raw = data.decode("utf-8", errors="replace").replace("\x00", "").strip()
            if not raw:
                continue

            parsed = parse_syslog_message(raw, source_ip)
            apply_registered_device_identity(
                parsed, device_id, source_ip, dev_hostname, dev_sys_name
            )

            buffer.append(parsed)

    except asyncio.TimeoutError:
        logger.debug("TCP syslog connection from %s timed out", source_ip)
    except Exception:
        logger.exception("Error processing TCP syslog from %s", source_ip)
    finally:
        writer.close()
        try:
            await writer.wait_closed()
        except Exception:
            pass


class SyslogReceiver:
    """Manages UDP and TCP syslog receivers with buffered writes and IP allowlisting."""

    def __init__(
        self,
        udp_port: int = 514,
        tcp_port: int = 514,
        allowlist_refresh_interval: int = ALLOWLIST_REFRESH_INTERVAL,
    ):
        self.udp_port = udp_port
        self.tcp_port = tcp_port
        self._buffer: list[dict[str, Any]] = []
        self._buffer_lock = asyncio.Lock()
        self._allowed_ips: dict[str, dict[str, Any]] = {}
        self._running = True
        self._flush_interval = 5  # seconds
        self._allowlist_refresh_interval = allowlist_refresh_interval

    def stop(self):
        self._running = False

    async def refresh_allowlist(self) -> None:
        """Fetch enabled device IPs and metadata from DB without blocking the event loop."""
        loop = asyncio.get_running_loop()
        raw_map = await loop.run_in_executor(None, db.get_allowed_devices)
        normalized = {normalize_ip(ip): dev_info for ip, dev_info in raw_map.items()}
        self._allowed_ips.clear()
        self._allowed_ips.update(normalized)

    async def _refresh_allowlist_loop(self) -> None:
        """Periodically refresh the in-memory IP allowlist from PostgreSQL."""
        while self._running:
            await asyncio.sleep(self._allowlist_refresh_interval)
            try:
                await self.refresh_allowlist()
                logger.debug("Refreshed syslog IP allowlist (%d devices)", len(self._allowed_ips))
            except Exception:
                logger.exception("Error refreshing syslog IP allowlist (keeping cached entries)")

    async def _flush_buffer(self) -> None:
        """Periodically flush buffered log messages to the database."""
        loop = asyncio.get_running_loop()
        while self._running:
            await asyncio.sleep(self._flush_interval)

            if not self._buffer:
                continue

            # Swap buffer to minimize lock time
            async with self._buffer_lock:
                to_flush = self._buffer[:]
                self._buffer.clear()

            if to_flush:
                try:
                    await loop.run_in_executor(None, db.insert_logs, to_flush)
                    logger.debug("Flushed %d log messages to database", len(to_flush))
                except Exception:
                    logger.exception("Error flushing logs to database")
                    # Re-add to buffer on failure (respecting cap)
                    async with self._buffer_lock:
                        space = MAX_BUFFER_SIZE - len(self._buffer)
                        if space > 0:
                            self._buffer.extend(to_flush[:space])

    async def run(self) -> None:
        """Start UDP and TCP syslog receivers."""
        loop = asyncio.get_running_loop()

        try:
            await self.refresh_allowlist()
            logger.info("Loaded syslog IP allowlist with %d enabled devices", len(self._allowed_ips))
        except Exception:
            logger.exception("Initial syslog allowlist load failed; will retry in background")

        # Start UDP server
        transport, protocol = await loop.create_datagram_endpoint(
            lambda: SyslogUDPProtocol(self._buffer, self._buffer_lock, self._allowed_ips),
            local_addr=("0.0.0.0", self.udp_port),
        )
        logger.info("Syslog UDP receiver listening on port %d", self.udp_port)

        # Start TCP server
        tcp_server = await asyncio.start_server(
            lambda r, w: handle_tcp_client(r, w, self._buffer, self._buffer_lock, self._allowed_ips),
            "0.0.0.0",
            self.tcp_port,
        )
        logger.info("Syslog TCP receiver listening on port %d", self.tcp_port)

        # Start background tasks
        flush_task = asyncio.create_task(self._flush_buffer())
        allowlist_task = asyncio.create_task(self._refresh_allowlist_loop())

        try:
            while self._running:
                await asyncio.sleep(1)
        finally:
            transport.close()
            tcp_server.close()
            await tcp_server.wait_closed()
            flush_task.cancel()
            allowlist_task.cancel()

            # Final flush
            if self._buffer:
                try:
                    await loop.run_in_executor(None, db.insert_logs, self._buffer)
                    logger.info("Final flush: %d log messages", len(self._buffer))
                except Exception:
                    logger.exception("Error in final log flush")
