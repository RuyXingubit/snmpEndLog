import pytest
import sys
import os

# Add parent directory to path so we can import snmp
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))

from snmp.parser import safe_int, safe_str, parse_mac_address, parse_uptime_ticks, calc_delta_bps
import time

def test_safe_int():
    assert safe_int("123") == 123
    assert safe_int(123) == 123
    assert safe_int("abc") == 0
    assert safe_int("abc", default=10) == 10

def test_safe_str():
    assert safe_str(b"hello") == "b'hello'" # byte string coerced to string by str()
    assert safe_str("hello") == "hello"
    assert safe_str(None) == ""

def test_parse_mac_address():
    assert parse_mac_address(b"\x00\x11\x22\x33\x44\x55") == "00:11:22:33:44:55"
    assert parse_mac_address("00:11:22:33:44:55") == "00:11:22:33:44:55"

def test_parse_uptime_ticks():
    assert parse_uptime_ticks(100) == 100
    assert parse_uptime_ticks("123") == 123
    assert parse_uptime_ticks(None) is None

def test_calc_delta_bps():
    # 1000 bytes difference over 1 second = 8000 bits per second
    bps = calc_delta_bps(
        current_octets=2000,
        previous_octets=1000,
        elapsed_seconds=1.0
    )
    assert bps == 8000.0

    # Test counter wrap-around 32-bit
    bps = calc_delta_bps(
        current_octets=100,
        previous_octets=4294967295,
        elapsed_seconds=1.0,
        counter_max=(2**32 - 1)
    )
    # wraps: formula uses (max - prev) + current
    assert bps == 100 * 8.0

