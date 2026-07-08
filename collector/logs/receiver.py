"""Syslog receiver — UDP and TCP servers for receiving syslog messages."""

import asyncio
import logging
from typing import Any

import db
from logs.parser import parse_syslog_message

logger = logging.getLogger(__name__)


class SyslogUDPProtocol(asyncio.DatagramProtocol):
    """UDP syslog receiver (RFC 3164/5424 over UDP)."""

    def __init__(self, buffer: list[dict[str, Any]], buffer_lock: asyncio.Lock):
        self._buffer = buffer
        self._buffer_lock = buffer_lock

    def connection_made(self, transport: asyncio.DatagramTransport) -> None:
        self._transport = transport

    def datagram_received(self, data: bytes, addr: tuple[str, int]) -> None:
        source_ip = addr[0]
        try:
            raw = data.decode("utf-8", errors="replace").strip()
            if not raw:
                return
            parsed = parse_syslog_message(raw, source_ip)

            # Try to resolve device_id
            device_id = db.resolve_device_id_by_ip(source_ip)
            parsed["device_id"] = device_id

            # Add to buffer (will be flushed periodically)
            # Use a simple append — the flush loop handles locking
            self._buffer.append(parsed)

        except Exception:
            logger.exception("Error processing UDP syslog from %s", source_ip)


async def handle_tcp_client(
    reader: asyncio.StreamReader,
    writer: asyncio.StreamWriter,
    buffer: list[dict[str, Any]],
    buffer_lock: asyncio.Lock,
) -> None:
    """Handle a single TCP syslog client connection."""
    addr = writer.get_extra_info("peername")
    source_ip = addr[0] if addr else "unknown"
    logger.debug("TCP syslog connection from %s", source_ip)

    try:
        while True:
            data = await asyncio.wait_for(reader.readline(), timeout=300)
            if not data:
                break

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

    def stop(self):
        self._running = False

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
                    # Re-add to buffer on failure
                    async with self._buffer_lock:
                        self._buffer.extend(to_flush)

    async def run(self) -> None:
        """Start UDP and TCP syslog receivers."""
        loop = asyncio.get_event_loop()

        # Start UDP server
        transport, protocol = await loop.create_datagram_endpoint(
            lambda: SyslogUDPProtocol(self._buffer, self._buffer_lock),
            local_addr=("0.0.0.0", self.udp_port),
        )
        logger.info("Syslog UDP receiver listening on port %d", self.udp_port)

        # Start TCP server
        tcp_server = await asyncio.start_server(
            lambda r, w: handle_tcp_client(r, w, self._buffer, self._buffer_lock),
            "0.0.0.0",
            self.tcp_port,
        )
        logger.info("Syslog TCP receiver listening on port %d", self.tcp_port)

        # Start buffer flusher
        flush_task = asyncio.create_task(self._flush_buffer())

        try:
            while self._running:
                await asyncio.sleep(1)
        finally:
            transport.close()
            tcp_server.close()
            await tcp_server.wait_closed()
            flush_task.cancel()

            # Final flush
            if self._buffer:
                try:
                    db.insert_logs(self._buffer)
                    logger.info("Final flush: %d log messages", len(self._buffer))
                except Exception:
                    logger.exception("Error in final log flush")
