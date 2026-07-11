"""Syslog receiver — UDP and TCP servers for receiving syslog messages.

Security features:
- IP allowlist: only accepts syslog from IPs registered as devices in the database
- Message size limit: 8192 bytes per message (RFC 5424)
- Buffer cap: 50,000 messages maximum to prevent memory exhaustion
- Silent drop of unauthorized IPs (no logging to prevent log flooding)
"""

import asyncio
import logging
import threading
from typing import Any

import db
from logs.parser import parse_syslog_message

logger = logging.getLogger(__name__)

# Security constants
MAX_MESSAGE_SIZE = 8192  # bytes, per RFC 5424
MAX_BUFFER_SIZE = 50_000  # maximum buffered messages before dropping
IP_CACHE_REFRESH_INTERVAL = 60  # seconds


class AllowedIPCache:
    """Thread-safe cache of allowed IPs, refreshed from the database periodically.

    Maintains a set of IP addresses from the `devices` table.
    IPs not in this set are silently rejected.
    """

    def __init__(self, refresh_interval: int = IP_CACHE_REFRESH_INTERVAL):
        self._allowed: set[str] = set()
        self._lock = threading.Lock()
        self._refresh_interval = refresh_interval

    def is_allowed(self, ip: str) -> bool:
        """Check if an IP is in the allowlist."""
        with self._lock:
            return ip in self._allowed

    def refresh(self) -> None:
        """Reload allowed IPs from the database."""
        try:
            with db.get_conn() as conn:
                with conn.cursor() as cur:
                    cur.execute("SELECT ip_address::text FROM devices")
                    new_ips = {row[0] for row in cur.fetchall()}

            with self._lock:
                added = new_ips - self._allowed
                removed = self._allowed - new_ips
                self._allowed = new_ips

            if added or removed:
                logger.info(
                    "IP allowlist updated: %d IPs (+%d -%d)",
                    len(new_ips), len(added), len(removed),
                )
        except Exception:
            logger.exception("Failed to refresh IP allowlist")


class SyslogUDPProtocol(asyncio.DatagramProtocol):
    """UDP syslog receiver (RFC 3164/5424 over UDP)."""

    def __init__(
        self,
        buffer: list[dict[str, Any]],
        buffer_lock: asyncio.Lock,
        ip_cache: AllowedIPCache,
    ):
        self._buffer = buffer
        self._buffer_lock = buffer_lock
        self._ip_cache = ip_cache

    def connection_made(self, transport: asyncio.DatagramTransport) -> None:
        self._transport = transport

    def datagram_received(self, data: bytes, addr: tuple[str, int]) -> None:
        source_ip = addr[0]

        # Security: reject IPs not in the allowlist (silent drop)
        if not self._ip_cache.is_allowed(source_ip):
            return

        # Security: reject oversized messages
        if len(data) > MAX_MESSAGE_SIZE:
            return

        # Security: reject if buffer is full
        if len(self._buffer) >= MAX_BUFFER_SIZE:
            return

        try:
            raw = data.decode("utf-8", errors="replace").strip()
            if not raw:
                return
            parsed = parse_syslog_message(raw, source_ip)

            # Resolve device_id (guaranteed to exist since IP is allowed)
            device_id = db.resolve_device_id_by_ip(source_ip)
            parsed["device_id"] = device_id

            self._buffer.append(parsed)

        except Exception:
            logger.exception("Error processing UDP syslog from %s", source_ip)


async def handle_tcp_client(
    reader: asyncio.StreamReader,
    writer: asyncio.StreamWriter,
    buffer: list[dict[str, Any]],
    buffer_lock: asyncio.Lock,
    ip_cache: AllowedIPCache,
) -> None:
    """Handle a single TCP syslog client connection."""
    addr = writer.get_extra_info("peername")
    source_ip = addr[0] if addr else "unknown"

    # Security: reject IPs not in the allowlist (silent drop, close connection)
    if not ip_cache.is_allowed(source_ip):
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

            # Security: reject oversized messages
            if len(data) > MAX_MESSAGE_SIZE:
                continue

            # Security: reject if buffer is full
            if len(buffer) >= MAX_BUFFER_SIZE:
                continue

            raw = data.decode("utf-8", errors="replace").strip()
            if not raw:
                continue

            parsed = parse_syslog_message(raw, source_ip)
            device_id = db.resolve_device_id_by_ip(source_ip)
            parsed["device_id"] = device_id

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
    """Manages UDP and TCP syslog receivers with buffered writes."""

    def __init__(self, udp_port: int = 514, tcp_port: int = 514):
        self.udp_port = udp_port
        self.tcp_port = tcp_port
        self._buffer: list[dict[str, Any]] = []
        self._buffer_lock = asyncio.Lock()
        self._running = True
        self._flush_interval = 5  # seconds
        self._ip_cache = AllowedIPCache()

    def stop(self):
        self._running = False

    async def _refresh_ip_cache(self) -> None:
        """Periodically refresh the IP allowlist from the database."""
        while self._running:
            # Run DB query in a thread to avoid blocking the event loop
            await asyncio.get_event_loop().run_in_executor(
                None, self._ip_cache.refresh
            )
            await asyncio.sleep(IP_CACHE_REFRESH_INTERVAL)

    async def _flush_buffer(self) -> None:
        """Periodically flush buffered log messages to the database."""
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
                    db.insert_logs(to_flush)
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
        loop = asyncio.get_event_loop()

        # Initial IP cache load
        await loop.run_in_executor(None, self._ip_cache.refresh)
        logger.info("IP allowlist loaded: %d devices", len(self._ip_cache._allowed))

        # Start UDP server
        transport, protocol = await loop.create_datagram_endpoint(
            lambda: SyslogUDPProtocol(self._buffer, self._buffer_lock, self._ip_cache),
            local_addr=("0.0.0.0", self.udp_port),
        )
        logger.info("Syslog UDP receiver listening on port %d", self.udp_port)

        # Start TCP server
        tcp_server = await asyncio.start_server(
            lambda r, w: handle_tcp_client(r, w, self._buffer, self._buffer_lock, self._ip_cache),
            "0.0.0.0",
            self.tcp_port,
        )
        logger.info("Syslog TCP receiver listening on port %d", self.tcp_port)

        # Start background tasks
        refresh_task = asyncio.create_task(self._refresh_ip_cache())
        flush_task = asyncio.create_task(self._flush_buffer())

        try:
            while self._running:
                await asyncio.sleep(1)
        finally:
            transport.close()
            tcp_server.close()
            await tcp_server.wait_closed()
            refresh_task.cancel()
            flush_task.cancel()

            # Final flush
            if self._buffer:
                try:
                    db.insert_logs(self._buffer)
                    logger.info("Final flush: %d log messages", len(self._buffer))
                except Exception:
                    logger.exception("Error in final log flush")
