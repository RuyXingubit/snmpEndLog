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

import db  # noqa: E402
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
            expected = {
                "177.53.139.255": {
                    "id": 12,
                    "hostname": "BGP-SJP",
                    "sys_name": "BGP-SJP",
                }
            }
            with patch("db.get_allowed_devices", return_value=expected):
                await receiver.refresh_allowlist()
            self.assertEqual(receiver._allowed_ips, expected)

        asyncio.run(_run())

    def test_allowed_device_host_binding_and_foreign_ip_sanitization(self):
        buffer = []
        lock = asyncio.Lock()
        allowed_ips = {
            "143.208.139.255": {
                "id": 7,
                "hostname": "BGP-ATM",
                "sys_name": "BNG-PROSERV",
            }
        }
        proto = SyslogUDPProtocol(buffer, lock, allowed_ips)

        # 1. Valid sys_name matching device identity keeps BNG-PROSERV
        proto.datagram_received(
            b"<134>Nov 16 12:00:00 BNG-PROSERV bgpd[10]: peer up",
            ("143.208.139.255", 514),
        )
        self.assertEqual(len(buffer), 1)
        self.assertEqual(buffer[0]["device_id"], 7)
        self.assertEqual(buffer[0]["host"], "BNG-PROSERV")

        # 2. Headless BSD message starting with a foreign IP rewrites host to registered identity
        #    and preserves the foreign IP inside message text
        proto.datagram_received(
            b"<134>Nov 16 12:00:00 188.240.59.45 login failure for root",
            ("143.208.139.255", 514),
        )
        self.assertEqual(len(buffer), 2)
        self.assertEqual(buffer[1]["device_id"], 7)
        self.assertEqual(buffer[1]["host"], "BNG-PROSERV")
        self.assertIn("188.240.59.45", buffer[1]["message"])

        # 3. Spoofed/unregistered hostname from allowed IP is normalized to registered identity
        proto.datagram_received(
            b"<134>Nov 16 12:00:00 evil-spoof sshd: msg",
            ("143.208.139.255", 514),
        )
        self.assertEqual(len(buffer), 3)
        self.assertEqual(buffer[2]["device_id"], 7)
        self.assertEqual(buffer[2]["host"], "BNG-PROSERV")
        self.assertEqual(buffer[2]["app_name"], "sshd")
        self.assertEqual(buffer[2]["message"], "msg")

    def test_insert_logs_skips_rows_without_device_id(self):
        mock_cursor = MagicMock()
        mock_conn = MagicMock()
        mock_conn.cursor.return_value.__enter__.return_value = mock_cursor
        mock_pool = MagicMock()
        mock_pool.getconn.return_value = mock_conn

        rows_all_orphan = [
            {"time": "2026-11-16T12:00:00Z", "host": "188.240.59.45", "device_id": None, "message": "scan"}
        ]
        rows_mixed = [
            {"time": "2026-11-16T12:00:00Z", "host": "188.240.59.45", "device_id": None, "message": "scan"},
            {"time": "2026-11-16T12:00:01Z", "host": "BNG-PROSERV", "device_id": 7, "message": "valid log"},
        ]

        with patch.object(db, "_pool", mock_pool), patch("db.psycopg2.extras.execute_values") as mock_exec:
            db.insert_logs(rows_all_orphan)
            mock_exec.assert_not_called()

            db.insert_logs(rows_mixed)
            mock_exec.assert_called_once()
            inserted_tuples = mock_exec.call_args[0][2]
            self.assertEqual(len(inserted_tuples), 1)
            self.assertEqual(inserted_tuples[0][1], "BNG-PROSERV")
            self.assertEqual(inserted_tuples[0][2], 7)


if __name__ == "__main__":
    unittest.main()
