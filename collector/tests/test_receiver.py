import asyncio
import os
import sys
import unittest
from unittest.mock import MagicMock, patch

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

# Mock psycopg2 before importing db / logs.receiver in environments without psycopg2 C-ext
sys.modules.setdefault("psycopg2", MagicMock())
sys.modules.setdefault("psycopg2.pool", MagicMock())
sys.modules.setdefault("psycopg2.extras", MagicMock())

from logs.receiver import (  # noqa: E402
    MAX_MESSAGE_SIZE,
    SyslogReceiver,
    SyslogUDPProtocol,
    handle_tcp_client,
    normalize_ip,
)


class TestSyslogReceiverSecurity(unittest.TestCase):
    def test_normalize_ip(self):
        self.assertEqual(normalize_ip("143.208.139.255"), "143.208.139.255")
        self.assertEqual(normalize_ip("::ffff:10.254.255.2"), "10.254.255.2")
        self.assertEqual(normalize_ip("2804:192c:0000:0000::1"), "2804:192c::1")

    def test_udp_drops_unauthorized_ip_and_accepts_allowed_ip(self):
        buffer = []
        lock = asyncio.Lock()
        allowed_ips = {"143.208.139.255": 7}

        proto = SyslogUDPProtocol(buffer, lock, allowed_ips)

        with patch("db.resolve_device_id_by_ip") as mock_db_lookup:
            # 1. Spoofed / external scanner IP must be silently dropped
            proto.datagram_received(
                b"<134>Nov 11 12:00:00 fakehost sshd: injected log",
                ("79.72.59.54", 51234),
            )
            self.assertEqual(len(buffer), 0)

            # 2. Registered device IP (even via IPv4-mapped IPv6) is accepted with O(1) device_id
            proto.datagram_received(
                b"<134>Nov 11 12:00:01 BGP-ATM bgpd: peer 177.53.136.1 Up",
                ("::ffff:143.208.139.255", 514),
            )
            self.assertEqual(len(buffer), 1)
            self.assertEqual(buffer[0]["device_id"], 7)
            self.assertIn("peer 177.53.136.1 Up", buffer[0]["message"])

            # 3. Oversized datagram from allowed IP is dropped
            proto.datagram_received(
                b"A" * (MAX_MESSAGE_SIZE + 1),
                ("143.208.139.255", 514),
            )
            self.assertEqual(len(buffer), 1)

            # Verify synchronous per-packet DB query is never called
            mock_db_lookup.assert_not_called()

    def test_tcp_closes_unauthorized_client_and_accepts_allowed_client(self):
        async def _run():
            buffer = []
            lock = asyncio.Lock()
            allowed_ips = {"10.254.255.2": 3}

            # 1. Unauthorized TCP peer is immediately closed without reading reader
            unauth_reader = asyncio.StreamReader()
            unauth_reader.feed_data(b"<134>Nov 11 12:00:00 attacker app: spoof\n")
            unauth_reader.feed_eof()

            unauth_writer = MagicMock()
            unauth_writer.get_extra_info.return_value = ("79.72.59.54", 44112)

            async def _noop_wait_closed():
                return None

            unauth_writer.wait_closed = _noop_wait_closed

            await handle_tcp_client(unauth_reader, unauth_writer, buffer, lock, allowed_ips)
            self.assertEqual(len(buffer), 0)
            unauth_writer.close.assert_called_once()

            # 2. Authorized TCP peer is accepted and buffered with device_id
            auth_reader = asyncio.StreamReader()
            auth_reader.feed_data(b"<134>Nov 11 12:00:02 SW-S6730 IFNET: GE0/0/1 UP\n")
            auth_reader.feed_eof()

            auth_writer = MagicMock()
            auth_writer.get_extra_info.return_value = ("10.254.255.2", 55110)
            auth_writer.wait_closed = _noop_wait_closed

            await handle_tcp_client(auth_reader, auth_writer, buffer, lock, allowed_ips)
            self.assertEqual(len(buffer), 1)
            self.assertEqual(buffer[0]["device_id"], 3)
            self.assertIn("GE0/0/1 UP", buffer[0]["message"])

        asyncio.run(_run())

    def test_receiver_refresh_allowlist(self):
        async def _run():
            receiver = SyslogReceiver(udp_port=5514, tcp_port=5514)
            with patch("db.get_allowed_device_ips", return_value={"177.53.139.255": 12}):
                await receiver.refresh_allowlist()
            self.assertEqual(receiver._allowed_ips, {"177.53.139.255": 12})

        asyncio.run(_run())


if __name__ == "__main__":
    unittest.main()
