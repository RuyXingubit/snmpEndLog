"""SNMP response parsing utilities."""

import logging

logger = logging.getLogger(__name__)

# 32-bit counter max for wrap detection
COUNTER32_MAX = 2**32 - 1
COUNTER64_MAX = 2**64 - 1


def safe_int(value, default: int = 0) -> int:
    """Safely convert an SNMP value to int."""
    try:
        return int(value)
    except (ValueError, TypeError):
        return default


def safe_str(value, default: str = "") -> str:
    """Safely convert an SNMP value to string, stripping NUL bytes."""
    try:
        if value is None:
            return default
        s = str(value)
        # Strip NUL bytes and filter non-printable characters
        s = s.replace("\x00", "")
        return "".join(c for c in s if c.isprintable() or c in ("\n", "\t"))
    except (ValueError, TypeError):
        return default


def parse_mac_address(value) -> str:
    """Parse a MAC address from SNMP OctetString."""
    try:
        if hasattr(value, "prettyPrint"):
            raw = value.prettyPrint()
            # Already formatted as XX:XX:XX:XX:XX:XX
            if ":" in raw and len(raw) == 17:
                return raw
        # Raw bytes
        if isinstance(value, (bytes, bytearray)):
            if len(value) == 6:
                return ":".join(f"{b:02x}" for b in value)
            return ""
        # Try OctetString with asNumbers()
        if hasattr(value, "asNumbers"):
            nums = value.asNumbers()
            if len(nums) == 6:
                return ":".join(f"{b:02x}" for b in nums)
            return ""
        # Hex string like 0x001122334455
        s = str(value).replace("0x", "").replace(":", "").replace("-", "").replace("\x00", "")
        if len(s) == 12:
            return ":".join(s[i : i + 2] for i in range(0, 12, 2))
        return ""
    except Exception:
        return ""


def calc_delta_bps(
    current_octets: int,
    previous_octets: int,
    elapsed_seconds: float,
    counter_max: int = COUNTER64_MAX,
) -> float | None:
    """Calculate bits per second from octet counter delta.

    Handles counter wraps gracefully.
    Returns None if calculation is invalid.
    """
    if elapsed_seconds <= 0 or current_octets is None or previous_octets is None:
        return None

    if current_octets >= previous_octets:
        delta = current_octets - previous_octets
    else:
        # Counter wrap
        delta = (counter_max - previous_octets) + current_octets
        logger.debug(
            "Counter wrap detected: current=%d, previous=%d, delta=%d",
            current_octets, previous_octets, delta,
        )

    bps = (delta * 8) / elapsed_seconds

    # Sanity check: if bps is absurdly high, likely a reset, not a wrap
    # Max realistic: 400 Gbps = 400_000_000_000
    if bps > 400_000_000_000:
        logger.warning(
            "Unrealistic bps value %.0f, likely counter reset. Returning None.",
            bps,
        )
        return None

    return bps


def parse_uptime_ticks(value) -> int | None:
    """Parse sysUpTime timeticks to integer."""
    try:
        return int(value)
    except (ValueError, TypeError):
        return None


def uptime_to_human(ticks: int | None) -> str:
    """Convert timeticks (hundredths of a second) to human-readable string."""
    if ticks is None:
        return "unknown"
    seconds = ticks // 100
    days, remainder = divmod(seconds, 86400)
    hours, remainder = divmod(remainder, 3600)
    minutes, secs = divmod(remainder, 60)
    parts = []
    if days > 0:
        parts.append(f"{days}d")
    if hours > 0:
        parts.append(f"{hours}h")
    if minutes > 0:
        parts.append(f"{minutes}m")
    parts.append(f"{secs}s")
    return " ".join(parts)


def extract_if_index(oid_str: str, base_oid: str) -> int | None:
    """Extract interface index from a full OID.

    Example: base='1.3.6.1.2.1.2.2.1.1', oid='1.3.6.1.2.1.2.2.1.1.5' -> 5
    """
    try:
        suffix = oid_str[len(base_oid) + 1 :]  # +1 for the dot
        return int(suffix.split(".")[0])
    except (ValueError, IndexError):
        return None
