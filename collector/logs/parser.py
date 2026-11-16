"""Syslog message parser supporting RFC 3164 (BSD) and RFC 5424."""

import ipaddress
import re
from datetime import datetime, timezone
from typing import Any

# ============================================
# Syslog Facility codes
# ============================================
FACILITY_NAMES = {
    0: "kern", 1: "user", 2: "mail", 3: "daemon",
    4: "auth", 5: "syslog", 6: "lpr", 7: "news",
    8: "uucp", 9: "cron", 10: "authpriv", 11: "ftp",
    12: "ntp", 13: "security", 14: "console", 15: "solaris-cron",
    16: "local0", 17: "local1", 18: "local2", 19: "local3",
    20: "local4", 21: "local5", 22: "local6", 23: "local7",
}

# ============================================
# Syslog Severity codes
# ============================================
SEVERITY_NAMES = {
    0: "emergency", 1: "alert", 2: "critical", 3: "error",
    4: "warning", 5: "notice", 6: "info", 7: "debug",
}

# ============================================
# RFC 3164 pattern: <PRI>TIMESTAMP HOSTNAME APP[PID]: MESSAGE
# Supports optional year (e.g. Huawei VRP: "Jul  9 2026 17:33:50")
# ============================================
RFC3164_RE = re.compile(
    r"<(\d{1,3})>"                          # PRI
    r"(\w{3}\s+\d{1,2}\s+(?:\d{4}\s+)?\d{2}:\d{2}:\d{2})\s+"  # Timestamp (with optional year)
    r"(\S+)\s+"                              # Hostname
    r"(.+)"                                   # Message (includes app name)
)

# ============================================
# RFC 5424 pattern: <PRI>VERSION TIMESTAMP HOSTNAME APP-NAME PROCID MSGID MSG
# ============================================
RFC5424_RE = re.compile(
    r"<(\d{1,3})>"                          # PRI
    r"(\d+)\s+"                              # VERSION
    r"(\S+)\s+"                              # TIMESTAMP (ISO 8601)
    r"(\S+)\s+"                              # HOSTNAME
    r"(\S+)\s+"                              # APP-NAME
    r"(\S+)\s+"                              # PROCID
    r"(\S+)\s*"                              # MSGID
    r"(.*)"                                   # MSG
)

# Month abbreviation to number
MONTH_MAP = {
    "Jan": 1, "Feb": 2, "Mar": 3, "Apr": 4,
    "May": 5, "Jun": 6, "Jul": 7, "Aug": 8,
    "Sep": 9, "Oct": 10, "Nov": 11, "Dec": 12,
}


def decode_priority(pri: int) -> tuple[int, int]:
    """Decode PRI value into facility and severity."""
    facility = pri >> 3
    severity = pri & 7
    return facility, severity


def parse_rfc3164_timestamp(ts_str: str) -> datetime:
    """Parse BSD syslog timestamp (Mmm dd HH:MM:SS).

    RFC 3164 timestamps lack timezone and year, which causes incorrect
    storage for devices in non-UTC timezones.  We use the server receive
    time instead, which is always accurate.
    """
    return datetime.now(timezone.utc)


def parse_syslog_message(raw: str, source_ip: str = "") -> dict[str, Any]:
    """Parse a syslog message (auto-detect RFC 3164 or 5424).

    Returns a dict with: time, host, facility, severity, facility_name,
    severity_name, app_name, message, raw
    """
    raw = raw.strip()

    # Try RFC 5424 first (has version field)
    match = RFC5424_RE.match(raw)
    if match:
        pri = int(match.group(1))
        facility, severity = decode_priority(pri)
        ts_str = match.group(3)
        hostname = match.group(4)
        app_name = match.group(5)
        message = match.group(8)

        # Parse ISO 8601 timestamp
        try:
            ts = datetime.fromisoformat(ts_str.replace("Z", "+00:00"))
        except ValueError:
            ts = datetime.now(timezone.utc)

        if hostname == "-":
            hostname = source_ip

        return {
            "time": ts,
            "host": hostname,
            "facility": facility,
            "severity": severity,
            "facility_name": FACILITY_NAMES.get(facility, f"facility{facility}"),
            "severity_name": SEVERITY_NAMES.get(severity, f"severity{severity}"),
            "app_name": app_name if app_name != "-" else None,
            "message": message,
            "raw": raw,
        }

    # Try RFC 3164 (BSD syslog)
    match = RFC3164_RE.match(raw)
    if match:
        pri = int(match.group(1))
        facility, severity = decode_priority(pri)
        ts = parse_rfc3164_timestamp(match.group(2))
        hostname = match.group(3)
        msg_part = match.group(4)

        # Try to extract app_name from message
        app_name = None
        app_match = re.match(r"(\S+?)(?:\[\d+\])?:\s*(.*)", msg_part)
        if app_match:
            app_name = app_match.group(1)
            message = app_match.group(2)
        else:
            message = msg_part

        return {
            "time": ts,
            "host": hostname,
            "facility": facility,
            "severity": severity,
            "facility_name": FACILITY_NAMES.get(facility, f"facility{facility}"),
            "severity_name": SEVERITY_NAMES.get(severity, f"severity{severity}"),
            "app_name": app_name,
            "message": message,
            "raw": raw,
        }

    # Fallback: unparseable message
    return {
        "time": datetime.now(timezone.utc),
        "host": source_ip or "unknown",
        "facility": None,
        "severity": None,
        "facility_name": None,
        "severity_name": None,
        "app_name": None,
        "message": raw,
        "raw": raw,
    }


def _canonical_ip_str(ip_str: str) -> str | None:
    """Return canonical IP string (unwrapping IPv4-mapped IPv6), or None if not an IP."""
    try:
        addr = ipaddress.ip_address(ip_str)
        if isinstance(addr, ipaddress.IPv6Address) and addr.ipv4_mapped:
            return str(addr.ipv4_mapped)
        return str(addr)
    except ValueError:
        return None


def _is_foreign_ip_or_tag(token: str, src_ip: str) -> bool:
    """Return True if token looks like a syslog app tag or an IP different from src_ip."""
    if not token:
        return False
    if token.endswith(":") or "[" in token:
        return True
    parsed_ip = _canonical_ip_str(token)
    if parsed_ip is None:
        return False
    norm_src = _canonical_ip_str(src_ip) or src_ip
    return parsed_ip != norm_src


def apply_registered_device_identity(
    parsed: dict[str, Any],
    device_id: int,
    source_ip: str,
    dev_hostname: str = "",
    dev_sys_name: str = "",
) -> dict[str, Any]:
    """Bind a parsed syslog entry to the verified registered device identity."""
    parsed["device_id"] = device_id

    clean_hostname = (dev_hostname or "").strip()
    clean_sys_name = (dev_sys_name or "").strip()
    raw_host = (parsed.get("host") or "").strip()

    allowed_names = {n.lower() for n in (clean_hostname, clean_sys_name) if n}
    canonical_host = clean_sys_name or clean_hostname or source_ip

    def _restore_misclassified_host_token() -> None:
        if raw_host.endswith(":") and parsed.get("app_name") is None:
            parsed["app_name"] = raw_host[:-1].split("[", 1)[0] or None
            return
        msg = parsed.get("message") or ""
        if not msg.startswith(raw_host):
            parsed["message"] = f"{raw_host} {msg}".strip()

    if allowed_names:
        if raw_host.lower() in allowed_names:
            parsed["host"] = raw_host
        else:
            if raw_host and _is_foreign_ip_or_tag(raw_host, source_ip):
                _restore_misclassified_host_token()
            parsed["host"] = canonical_host
    else:
        if raw_host and _is_foreign_ip_or_tag(raw_host, source_ip):
            _restore_misclassified_host_token()
            parsed["host"] = source_ip

    return parsed
