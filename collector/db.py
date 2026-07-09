"""Database connection pool and batch insert helpers."""

import logging
from contextlib import contextmanager
from datetime import datetime, timezone
from typing import Any

import psycopg2
import psycopg2.extras
import psycopg2.pool

from config import Config

logger = logging.getLogger(__name__)

_pool: psycopg2.pool.ThreadedConnectionPool | None = None


def init_pool(min_conn: int = 2, max_conn: int = 10) -> None:
    """Initialize the database connection pool."""
    global _pool
    _pool = psycopg2.pool.ThreadedConnectionPool(
        min_conn, max_conn, Config.dsn()
    )
    logger.info("Database connection pool initialized (min=%d, max=%d)", min_conn, max_conn)


def close_pool() -> None:
    """Close all connections in the pool."""
    global _pool
    if _pool:
        _pool.closeall()
        _pool = None
        logger.info("Database connection pool closed")


@contextmanager
def get_conn():
    """Get a connection from the pool (context manager)."""
    if not _pool:
        raise RuntimeError("Database pool not initialized. Call init_pool() first.")
    conn = _pool.getconn()
    try:
        yield conn
        conn.commit()
    except Exception:
        conn.rollback()
        raise
    finally:
        _pool.putconn(conn)


# ---- Device operations ----

def get_enabled_devices() -> list[dict[str, Any]]:
    """Fetch all enabled devices from the database."""
    with get_conn() as conn:
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("""
                SELECT id, hostname, ip_address::text as ip_address,
                       snmp_version, community,
                       snmpv3_user, snmpv3_auth_proto, snmpv3_auth_pass,
                       snmpv3_priv_proto, snmpv3_priv_pass, snmpv3_sec_level,
                       poll_interval, ping_enabled, snmp_enabled, sys_name
                FROM devices
                WHERE enabled = TRUE
            """)
            return [dict(row) for row in cur.fetchall()]


def update_device_info(device_id: int, info: dict[str, Any]) -> None:
    """Update device system information after a poll."""
    with get_conn() as conn:
        with conn.cursor() as cur:
            cur.execute("""
                UPDATE devices SET
                    sys_descr = %(sys_descr)s,
                    sys_object_id = COALESCE(%(sys_object_id)s, sys_object_id),
                    sys_name = %(sys_name)s,
                    sys_location = %(sys_location)s,
                    sys_contact = %(sys_contact)s,
                    sys_uptime = %(sys_uptime)s,
                    voltage = %(voltage)s,
                    vendor = COALESCE(%(vendor)s, vendor),
                    board_name = COALESCE(%(board_name)s, board_name),
                    serial_number = COALESCE(%(serial_number)s, serial_number),
                    firmware_version = COALESCE(%(firmware_version)s, firmware_version),
                    status = %(status)s,
                    last_polled_at = %(last_polled_at)s,
                    last_seen_at = %(last_seen_at)s,
                    updated_at = NOW()
                WHERE id = %(device_id)s
            """, {
                "sys_descr": info.get("sys_descr"),
                "sys_object_id": info.get("sys_object_id"),
                "sys_name": info.get("sys_name"),
                "sys_location": info.get("sys_location"),
                "sys_contact": info.get("sys_contact"),
                "sys_uptime": info.get("sys_uptime"),
                "voltage": info.get("voltage"),
                "vendor": info.get("vendor"),
                "board_name": info.get("board_name"),
                "serial_number": info.get("serial_number"),
                "firmware_version": info.get("firmware_version"),
                "status": info.get("status"),
                "last_polled_at": info.get("last_polled_at"),
                "last_seen_at": info.get("last_seen_at"),
                "device_id": device_id,
            })


def update_device_status(device_id: int, status: str) -> None:
    """Update device status (up/down/unknown)."""
    now = datetime.now(timezone.utc)
    with get_conn() as conn:
        with conn.cursor() as cur:
            params: dict[str, Any] = {
                "status": status,
                "device_id": device_id,
                "last_polled_at": now,
            }
            if status == "up":
                cur.execute("""
                    UPDATE devices SET status = %(status)s,
                        last_polled_at = %(last_polled_at)s,
                        last_seen_at = %(last_polled_at)s,
                        updated_at = NOW()
                    WHERE id = %(device_id)s
                """, params)
            else:
                cur.execute("""
                    UPDATE devices SET status = %(status)s,
                        last_polled_at = %(last_polled_at)s,
                        updated_at = NOW()
                    WHERE id = %(device_id)s
                """, params)


# ---- Interface operations ----

def upsert_interfaces(device_id: int, interfaces: list[dict[str, Any]]) -> None:
    """Insert or update discovered interfaces for a device."""
    with get_conn() as conn:
        with conn.cursor() as cur:
            for iface in interfaces:
                cur.execute("""
                    INSERT INTO interfaces
                        (device_id, if_index, if_descr, if_alias, if_type,
                         if_speed, if_hc_speed, if_admin_status, if_oper_status,
                         if_phys_address, vlan_type, native_vlan, updated_at)
                    VALUES
                        (%(device_id)s, %(if_index)s, %(if_descr)s, %(if_alias)s,
                         %(if_type)s, %(if_speed)s, %(if_hc_speed)s,
                         %(if_admin_status)s, %(if_oper_status)s,
                         %(if_phys_address)s, %(vlan_type)s, %(native_vlan)s, NOW())
                    ON CONFLICT (device_id, if_index) DO UPDATE SET
                        if_descr = EXCLUDED.if_descr,
                        if_alias = EXCLUDED.if_alias,
                        if_type = EXCLUDED.if_type,
                        if_speed = EXCLUDED.if_speed,
                        if_hc_speed = EXCLUDED.if_hc_speed,
                        if_admin_status = EXCLUDED.if_admin_status,
                        if_oper_status = EXCLUDED.if_oper_status,
                        if_phys_address = EXCLUDED.if_phys_address,
                        vlan_type = EXCLUDED.vlan_type,
                        native_vlan = EXCLUDED.native_vlan,
                        updated_at = NOW()
                """, {**iface, "device_id": device_id})


def upsert_bgp_peers(device_id: int, peers: list[dict[str, Any]]) -> None:
    """Insert or update current BGP peer status for a device."""
    if not peers:
        return
    with get_conn() as conn:
        with conn.cursor() as cur:
            # Delete old peers that are not in the new list?
            # Or just update them and let them age out. 
            # Given we have a UI, it's better to clean up deleted peers or we can just leave them if they don't respond.
            # We'll just upsert for now.
            psycopg2.extras.execute_values(
                cur,
                """
                INSERT INTO bgp_peers 
                    (device_id, peer_addr, peer_as, state, admin_status, 
                     in_updates, out_updates, prefixes_received, prefixes_advertised, fsm_established_time, updated_at)
                VALUES %s
                ON CONFLICT (device_id, peer_addr) DO UPDATE SET
                    peer_as = EXCLUDED.peer_as,
                    state = EXCLUDED.state,
                    admin_status = EXCLUDED.admin_status,
                    in_updates = EXCLUDED.in_updates,
                    out_updates = EXCLUDED.out_updates,
                    prefixes_received = EXCLUDED.prefixes_received,
                    prefixes_advertised = EXCLUDED.prefixes_advertised,
                    fsm_established_time = EXCLUDED.fsm_established_time,
                    updated_at = EXCLUDED.updated_at
                """,
                [
                    (
                        r["device_id"], r["peer_addr"], r.get("peer_as"),
                        r.get("state"), r.get("admin_status"),
                        r.get("in_updates"), r.get("out_updates"),
                        r.get("prefixes_received"), r.get("prefixes_advertised"),
                        r.get("fsm_established_time"), r["updated_at"]
                    )
                    for r in peers
                ]
            )

# ---- Metric inserts (batch) ----

def insert_traffic_metrics(rows: list[dict[str, Any]]) -> None:
    """Batch insert traffic metrics."""
    if not rows:
        return
    with get_conn() as conn:
        with conn.cursor() as cur:
            psycopg2.extras.execute_values(
                cur,
                """INSERT INTO metric_traffic
                   (time, device_id, if_index, in_octets, out_octets,
                    in_bps, out_bps, in_errors, out_errors)
                   VALUES %s""",
                [
                    (
                        r["time"], r["device_id"], r["if_index"],
                        r.get("in_octets"), r.get("out_octets"),
                        r.get("in_bps"), r.get("out_bps"),
                        r.get("in_errors", 0), r.get("out_errors", 0),
                    )
                    for r in rows
                ],
            )


def insert_system_metrics(rows: list[dict[str, Any]]) -> None:
    """Batch insert system metrics."""
    if not rows:
        return
    with get_conn() as conn:
        with conn.cursor() as cur:
            psycopg2.extras.execute_values(
                cur,
                """INSERT INTO metric_system
                   (time, device_id, cpu_percent, memory_percent,
                    memory_used, memory_total, uptime, pppoe_online, temperature)
                   VALUES %s""",
                [
                    (
                        r["time"], r["device_id"],
                        r.get("cpu_percent"), r.get("memory_percent"),
                        r.get("memory_used"), r.get("memory_total"),
                        r.get("uptime"), r.get("pppoe_online"),
                        r.get("temperature"),
                    )
                    for r in rows
                ],
            )


def insert_ping_metrics(rows: list[dict[str, Any]]) -> None:
    """Batch insert ping metrics."""
    if not rows:
        return
    with get_conn() as conn:
        with conn.cursor() as cur:
            psycopg2.extras.execute_values(
                cur,
                """INSERT INTO metric_ping
                   (time, device_id, rtt_min, rtt_avg, rtt_max,
                    packet_loss, is_reachable)
                   VALUES %s""",
                [
                    (
                        r["time"], r["device_id"],
                        r.get("rtt_min"), r.get("rtt_avg"), r.get("rtt_max"),
                        r.get("packet_loss", 0), r.get("is_reachable", True),
                    )
                    for r in rows
                ],
            )


# ---- Log inserts (batch) ----

def insert_logs(rows: list[dict[str, Any]]) -> None:
    """Batch insert syslog messages."""
    if not rows:
        return
    with get_conn() as conn:
        with conn.cursor() as cur:
            psycopg2.extras.execute_values(
                cur,
                """INSERT INTO logs
                   (time, host, device_id, facility, severity,
                    facility_name, severity_name, app_name, message, raw)
                   VALUES %s""",
                [
                    (
                        r["time"], r["host"], r.get("device_id"),
                        r.get("facility"), r.get("severity"),
                        r.get("facility_name"), r.get("severity_name"),
                        r.get("app_name"), r["message"], r.get("raw"),
                    )
                    for r in rows
                ],
            )


def resolve_device_id_by_ip(ip: str) -> int | None:
    """Look up a device ID by its IP address."""
    with get_conn() as conn:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT id FROM devices WHERE ip_address = %s::inet",
                (ip,),
            )
            row = cur.fetchone()
            return row[0] if row else None


# ---- BGP Metrics (batch) ----

def insert_bgp_metrics(rows: list[dict[str, Any]]) -> None:
    """Batch insert BGP metrics."""
    if not rows:
        return
    with get_conn() as conn:
        with conn.cursor() as cur:
            psycopg2.extras.execute_values(
                cur,
                """INSERT INTO metric_bgp
                   (time, device_id, peer_addr, state, uptime)
                   VALUES %s""",
                [
                    (
                        r["time"], r["device_id"], r["peer_addr"],
                        r.get("state"), r.get("uptime")
                    )
                    for r in rows
                ],
            )

# ---- Alarms ----

def set_alarm(device_id: int, entity_type: str, entity_id: str, name: str, severity: str, message: str) -> None:
    """Create or update an active alarm."""
    with get_conn() as conn:
        with conn.cursor() as cur:
            cur.execute("""
                INSERT INTO alarms (device_id, entity_type, entity_id, name, severity, message, status, created_at)
                VALUES (%(device_id)s, %(entity_type)s, %(entity_id)s, %(name)s, %(severity)s, %(message)s, 'active', NOW())
                ON CONFLICT (device_id, entity_type, entity_id) WHERE status = 'active'
                DO UPDATE SET
                    name = EXCLUDED.name,
                    severity = EXCLUDED.severity,
                    message = EXCLUDED.message
            """, {
                "device_id": device_id,
                "entity_type": entity_type,
                "entity_id": entity_id,
                "name": name,
                "severity": severity,
                "message": message
            })

def resolve_alarm(device_id: int, entity_type: str, entity_id: str) -> None:
    """Resolve an active alarm."""
    with get_conn() as conn:
        with conn.cursor() as cur:
            cur.execute("""
                UPDATE alarms 
                SET status = 'resolved', resolved_at = NOW()
                WHERE device_id = %(device_id)s 
                  AND entity_type = %(entity_type)s 
                  AND entity_id = %(entity_id)s 
                  AND status = 'active'
            """, {
                "device_id": device_id,
                "entity_type": entity_type,
                "entity_id": entity_id
            })

def get_interface_states(device_id: int) -> dict[int, int]:
    """Fetch current ifOperStatus for all interfaces of a device to compare against new polling results."""
    with get_conn() as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT if_index, if_oper_status FROM interfaces WHERE device_id = %s", (device_id,))
            return {row[0]: row[1] for row in cur.fetchall() if row[1] is not None}
