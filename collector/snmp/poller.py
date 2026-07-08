"""SNMP polling engine with support for SNMPv2c and SNMPv3."""

import asyncio
import logging
import subprocess
import time
from datetime import datetime, timezone
from typing import Any

from pysnmp.hlapi.v3arch.asyncio import (
    CommunityData,
    ContextData,
    ObjectIdentity,
    ObjectType,
    SnmpEngine,
    UdpTransportTarget,
    UsmUserData,
    bulk_cmd,
    get_cmd,
)
from pysnmp.hlapi.v3arch.asyncio import (
    usmHMACMD5AuthProtocol,
    usmHMACSHAAuthProtocol,
    usmHMAC128SHA224AuthProtocol,
    usmHMAC192SHA256AuthProtocol,
    usmHMAC256SHA384AuthProtocol,
    usmHMAC384SHA512AuthProtocol,
    usmDESPrivProtocol,
    usmAesCfb128Protocol,
    usmAesCfb256Protocol,
)

import db
from snmp.oids import (
    HostResourcesOIDs,
    InterfaceOIDs,
    MikroTikOIDs,
    HuaweiOIDs,
    SystemOIDs,
    Bgp4OIDs,
    INTERFACE_HC_WALK_OIDS,
    detect_vendor,
)
from snmp.parser import (
    calc_delta_bps,
    extract_if_index,
    parse_mac_address,
    parse_uptime_ticks,
    safe_int,
    safe_str,
)

logger = logging.getLogger(__name__)

# Auth/Priv protocol mapping
AUTH_PROTOCOLS = {
    "MD5": usmHMACMD5AuthProtocol,
    "SHA": usmHMACSHAAuthProtocol,
    "SHA224": usmHMAC128SHA224AuthProtocol,
    "SHA256": usmHMAC192SHA256AuthProtocol,
    "SHA384": usmHMAC256SHA384AuthProtocol,
    "SHA512": usmHMAC384SHA512AuthProtocol,
}

PRIV_PROTOCOLS = {
    "DES": usmDESPrivProtocol,
    "AES": usmAesCfb128Protocol,
    "AES128": usmAesCfb128Protocol,
    "AES256": usmAesCfb256Protocol,
}

# Store previous counter values for delta calculation
_previous_counters: dict[str, dict[str, Any]] = {}


def clean_ip(ip: str) -> str:
    """Strip CIDR notation from IP address (e.g., '10.0.0.1/24' -> '10.0.0.1')."""
    return ip.split("/")[0].strip()


class SNMPPoller:
    """Polls SNMP devices and stores metrics."""

    def __init__(self):
        self.engine = SnmpEngine()
        self._running = True

    def stop(self):
        self._running = False

    def _get_auth_data(self, device: dict) -> CommunityData | UsmUserData:
        """Build SNMP authentication data from device config."""
        if device["snmp_version"] == "v2c":
            return CommunityData(device.get("community", "public"))

        # SNMPv3
        sec_level = device.get("snmpv3_sec_level", "authPriv")
        user = device.get("snmpv3_user", "")

        if sec_level == "noAuthNoPriv":
            return UsmUserData(user)
        elif sec_level == "authNoPriv":
            auth_proto = AUTH_PROTOCOLS.get(
                device.get("snmpv3_auth_proto", "SHA"),
                usmHMACSHAAuthProtocol,
            )
            return UsmUserData(
                user,
                device.get("snmpv3_auth_pass", ""),
                authProtocol=auth_proto,
            )
        else:  # authPriv
            auth_proto = AUTH_PROTOCOLS.get(
                device.get("snmpv3_auth_proto", "SHA"),
                usmHMACSHAAuthProtocol,
            )
            priv_proto = PRIV_PROTOCOLS.get(
                device.get("snmpv3_priv_proto", "AES"),
                usmAesCfb128Protocol,
            )
            return UsmUserData(
                user,
                device.get("snmpv3_auth_pass", ""),
                device.get("snmpv3_priv_pass", ""),
                authProtocol=auth_proto,
                privProtocol=priv_proto,
            )

    async def _snmp_get(
        self, device: dict, *oids: str
    ) -> dict[str, Any]:
        """Perform SNMP GET for specific OIDs."""
        auth_data = self._get_auth_data(device)
        transport = await UdpTransportTarget.create((clean_ip(device["ip_address"]), 161))

        result = {}
        error_indication, error_status, error_index, var_binds = await get_cmd(
            self.engine,
            auth_data,
            transport,
            ContextData(),
            *[ObjectType(ObjectIdentity(oid)) for oid in oids],
        )

        if error_indication:
            logger.warning(
                "SNMP GET error for %s: %s", device["hostname"], error_indication
            )
            return result

        if error_status:
            logger.warning(
                "SNMP GET error status for %s: %s at %s",
                device["hostname"],
                error_status.prettyPrint(),
                error_index and var_binds[int(error_index) - 1][0] or "?",
            )
            return result

        for oid, val in var_binds:
            result[str(oid)] = val
        return result

    async def _snmp_walk(
        self, device: dict, base_oid: str
    ) -> list[tuple[str, Any]]:
        """Perform SNMP BULK WALK for a table OID using repeated bulk_cmd calls."""
        auth_data = self._get_auth_data(device)
        ip = clean_ip(device["ip_address"])

        results = []
        next_oid = base_oid

        while True:
            transport = await UdpTransportTarget.create((ip, 161))
            error_indication, error_status, error_index, var_binds = await bulk_cmd(
                self.engine,
                auth_data,
                transport,
                ContextData(),
                0,  # nonRepeaters
                25,  # maxRepetitions
                ObjectType(ObjectIdentity(next_oid)),
                lookupMib=False,
            )

            if error_indication:
                logger.warning(
                    "SNMP WALK error for %s oid %s: %s",
                    device["hostname"], base_oid, error_indication,
                )
                break

            if error_status:
                break

            if not var_binds:
                break

            done = False
            for row in var_binds:
                oid, val = row
                oid_str = str(oid)
                if not oid_str.startswith(base_oid):
                    done = True
                    break
                results.append((oid_str, val))
                next_oid = oid_str

            if done or len(var_binds) == 0:
                break

        return results

    async def poll_system_info(self, device: dict) -> dict[str, Any]:
        """Poll system MIB information and detect vendor."""
        result = await self._snmp_get(
            device,
            SystemOIDs.SYS_DESCR,
            SystemOIDs.SYS_OBJECT,
            SystemOIDs.SYS_NAME,
            SystemOIDs.SYS_LOCATION,
            SystemOIDs.SYS_CONTACT,
            SystemOIDs.SYS_UPTIME,
        )

        if not result:
            return {}

        now = datetime.now(timezone.utc)
        sys_object_id = safe_str(result.get(SystemOIDs.SYS_OBJECT))
        vendor = detect_vendor(sys_object_id)

        info = {
            "sys_descr": safe_str(result.get(SystemOIDs.SYS_DESCR)),
            "sys_object_id": sys_object_id,
            "sys_name": safe_str(result.get(SystemOIDs.SYS_NAME)),
            "sys_location": safe_str(result.get(SystemOIDs.SYS_LOCATION)),
            "sys_contact": safe_str(result.get(SystemOIDs.SYS_CONTACT)),
            "sys_uptime": parse_uptime_ticks(result.get(SystemOIDs.SYS_UPTIME)),
            "vendor": vendor,
            "status": "up",
            "last_polled_at": now,
            "last_seen_at": now,
        }

        # MikroTik-specific enrichment
        if vendor == "MikroTik":
            mk_result = await self._snmp_get(
                device,
                MikroTikOIDs.SERIAL_NUMBER,
                MikroTikOIDs.FIRMWARE_VERSION,
                MikroTikOIDs.BOARD_NAME,
                MikroTikOIDs.HL_VOLTAGE,
            )
            if mk_result:
                info["serial_number"] = safe_str(mk_result.get(MikroTikOIDs.SERIAL_NUMBER))
                info["firmware_version"] = safe_str(mk_result.get(MikroTikOIDs.FIRMWARE_VERSION))
                info["board_name"] = safe_str(mk_result.get(MikroTikOIDs.BOARD_NAME))
                
                # Voltage is returned in tenths of a volt (e.g. 245 = 24.5V)
                voltage_raw = safe_int(mk_result.get(MikroTikOIDs.HL_VOLTAGE))
                if voltage_raw is not None:
                    info["voltage"] = voltage_raw / 10.0
                else:
                    info["voltage"] = None
                logger.info(
                    "MikroTik detected: %s (board=%s, fw=%s, sn=%s)",
                    device["hostname"],
                    info.get("board_name", "?"),
                    info.get("firmware_version", "?"),
                    info.get("serial_number", "?"),
                )

        # Huawei-specific enrichment
        elif vendor == "Huawei":
            # Serial Number is scalar
            hw_res = await self._snmp_get(device, HuaweiOIDs.SERIAL_NUMBER)
            if hw_res and hw_res.get(HuaweiOIDs.SERIAL_NUMBER):
                info["serial_number"] = safe_str(hw_res.get(HuaweiOIDs.SERIAL_NUMBER))
                
            # Board Name (walk and take first valid)
            board_rows = await self._snmp_walk(device, HuaweiOIDs.HW_BOARD_NAME)
            if board_rows:
                for _, b_name in board_rows:
                    b_str = safe_str(b_name)
                    if b_str:
                        info["board_name"] = b_str
                        break
                        
            # Voltage (walk and take max, unit is mV)
            volt_rows = await self._snmp_walk(device, HuaweiOIDs.HW_ENTITY_VOLTAGE)
            if volt_rows:
                volts = [safe_int(v) for _, v in volt_rows if safe_int(v)]
                if volts:
                    # Convert max mV to V
                    info["voltage"] = max(volts) / 1000.0

        return info

    async def poll_interfaces(self, device: dict) -> list[dict[str, Any]]:
        """Discover and poll interface information."""
        interfaces: dict[int, dict[str, Any]] = {}

        # Walk ifTable (32-bit)
        for oid in [
            InterfaceOIDs.IF_INDEX,
            InterfaceOIDs.IF_DESCR,
            InterfaceOIDs.IF_TYPE,
            InterfaceOIDs.IF_SPEED,
            InterfaceOIDs.IF_PHYS_ADDR,
            InterfaceOIDs.IF_ADMIN_STATUS,
            InterfaceOIDs.IF_OPER_STATUS,
            InterfaceOIDs.IF_IN_OCTETS,
            InterfaceOIDs.IF_IN_ERRORS,
            InterfaceOIDs.IF_OUT_OCTETS,
            InterfaceOIDs.IF_OUT_ERRORS,
        ]:
            rows = await self._snmp_walk(device, oid)
            for oid_str, val in rows:
                idx = extract_if_index(oid_str, oid)
                if idx is None:
                    continue
                if idx not in interfaces:
                    interfaces[idx] = {"if_index": idx}
                field_map = {
                    InterfaceOIDs.IF_INDEX: ("if_index", safe_int),
                    InterfaceOIDs.IF_DESCR: ("if_descr", safe_str),
                    InterfaceOIDs.IF_TYPE: ("if_type", safe_int),
                    InterfaceOIDs.IF_SPEED: ("if_speed", safe_int),
                    InterfaceOIDs.IF_PHYS_ADDR: ("if_phys_address", parse_mac_address),
                    InterfaceOIDs.IF_ADMIN_STATUS: ("if_admin_status", safe_int),
                    InterfaceOIDs.IF_OPER_STATUS: ("if_oper_status", safe_int),
                    InterfaceOIDs.IF_IN_OCTETS: ("in_octets_32", safe_int),
                    InterfaceOIDs.IF_IN_ERRORS: ("in_errors", safe_int),
                    InterfaceOIDs.IF_OUT_OCTETS: ("out_octets_32", safe_int),
                    InterfaceOIDs.IF_OUT_ERRORS: ("out_errors", safe_int),
                }
                if oid in field_map:
                    field_name, parser = field_map[oid]
                    interfaces[idx][field_name] = parser(val)

        # Walk ifXTable (64-bit counters — preferred)
        for hc_oid in INTERFACE_HC_WALK_OIDS:
            rows = await self._snmp_walk(device, hc_oid)
            for oid_str, val in rows:
                idx = extract_if_index(oid_str, hc_oid)
                if idx is None or idx not in interfaces:
                    continue
                hc_map = {
                    InterfaceOIDs.IF_NAME: ("if_name", safe_str),
                    InterfaceOIDs.IF_HC_IN_OCTETS: ("in_octets_64", safe_int),
                    InterfaceOIDs.IF_HC_OUT_OCTETS: ("out_octets_64", safe_int),
                    InterfaceOIDs.IF_HIGH_SPEED: ("if_hc_speed", safe_int),
                    InterfaceOIDs.IF_ALIAS: ("if_alias", safe_str),
                }
                if hc_oid in hc_map:
                    field_name, parser = hc_map[hc_oid]
                    interfaces[idx][field_name] = parser(val)

        # Filter out PPPoE / Virtual interfaces to avoid bloating the DB
        filtered_interfaces = []
        pppoe_count = 0
        for iface in interfaces.values():
            name = iface.get("if_name", "").lower()
            descr = iface.get("if_descr", "").lower()
            alias = iface.get("if_alias", "").lower()
            combined = f"{name} {descr} {alias}"
            
            # Count MikroTik PPPoE before filtering
            if iface.get("if_type") == 23 and iface.get("if_oper_status") == 1:
                pppoe_count += 1
                
            # Skip if it looks like a PPPoE or Virtual Access interface
            if "<pppoe" in combined or "virtual-access" in combined or "virtual-template" in combined:
                continue
            
            filtered_interfaces.append(iface)
            
        device["_pppoe_count"] = pppoe_count

        return filtered_interfaces

    async def poll_host_resources(self, device: dict) -> dict[str, Any]:
        """Poll CPU and memory from Host Resources MIB, or vendor-specific MIBs."""
        result: dict[str, Any] = {"cpu_percent": None, "memory_percent": None,
                                   "memory_used": None, "memory_total": None}
                                   
        vendor = device.get("vendor", "")
        logger.info(f"poll_host_resources for {device['hostname']}: vendor is '{vendor}'")

        # Huawei-specific CPU/Mem
        if vendor == "Huawei":
            # CPU (walk hwEntityCpuUsage and take max or average)
            cpu_rows = await self._snmp_walk(device, HuaweiOIDs.HW_CPU_USAGE)
            if cpu_rows:
                loads = [safe_int(v) for _, v in cpu_rows if safe_int(v) is not None]
                if loads:
                    result["cpu_percent"] = max(loads)  # Or sum(loads)/len(loads)
                    
            # Memory (walk hwEntityMemUsage)
            mem_rows = await self._snmp_walk(device, HuaweiOIDs.HW_MEMORY_USAGE)
            if mem_rows:
                mems = [safe_int(v) for _, v in mem_rows if safe_int(v) is not None]
                if mems:
                    result["memory_percent"] = max(mems)
                    
            # PPPoE Online Users (Scalar)
            pppoe_res = await self._snmp_get(device, HuaweiOIDs.PPPOE_ONLINE)
            if pppoe_res and pppoe_res.get(HuaweiOIDs.PPPOE_ONLINE):
                result["pppoe_online"] = safe_int(pppoe_res.get(HuaweiOIDs.PPPOE_ONLINE))
            
            logger.info(f"Huawei polled: CPU rows={len(cpu_rows) if cpu_rows else 0} MEM rows={len(mem_rows) if mem_rows else 0} PPPoE res={pppoe_res}")
                    
            # If Huawei returned valid specific data, return it
            if result["cpu_percent"] is not None or result["memory_percent"] is not None or "pppoe_online" in result:
                return result

        # CPU — average of all processor loads
        cpu_rows = await self._snmp_walk(device, HostResourcesOIDs.HR_PROCESSOR_LOAD)
        if cpu_rows:
            loads = [safe_int(val) for _, val in cpu_rows]
            result["cpu_percent"] = sum(loads) / len(loads) if loads else None

        # Memory — find RAM storage entry
        descr_rows = await self._snmp_walk(device, HostResourcesOIDs.HR_STORAGE_DESCR)
        type_rows = await self._snmp_walk(device, HostResourcesOIDs.HR_STORAGE_TYPE)
        alloc_rows = await self._snmp_walk(device, HostResourcesOIDs.HR_STORAGE_ALLOC)
        size_rows = await self._snmp_walk(device, HostResourcesOIDs.HR_STORAGE_SIZE)
        used_rows = await self._snmp_walk(device, HostResourcesOIDs.HR_STORAGE_USED)

        # Build storage index map
        def _idx(oid_str: str, base: str) -> str:
            return oid_str[len(base):]

        alloc_map = {_idx(o, HostResourcesOIDs.HR_STORAGE_ALLOC): safe_int(v, 1) for o, v in alloc_rows}
        size_map = {_idx(o, HostResourcesOIDs.HR_STORAGE_SIZE): safe_int(v) for o, v in size_rows}
        used_map = {_idx(o, HostResourcesOIDs.HR_STORAGE_USED): safe_int(v) for o, v in used_rows}
        type_map = {_idx(o, HostResourcesOIDs.HR_STORAGE_TYPE): safe_str(v) for o, v in type_rows}

        for idx, type_val in type_map.items():
            if HostResourcesOIDs.HR_STORAGE_RAM in str(type_val):
                alloc = alloc_map.get(idx, 1)
                total = size_map.get(idx, 0) * alloc
                used = used_map.get(idx, 0) * alloc
                if total > 0:
                    result["memory_total"] = total
                    result["memory_used"] = used
                    result["memory_percent"] = (used / total) * 100
                break

        return result

    async def ping_device(self, device: dict) -> dict[str, Any]:
        """Ping a device and return latency metrics."""
        ip = clean_ip(device["ip_address"])
        try:
            proc = await asyncio.create_subprocess_exec(
                "ping", "-c", "3", "-W", "2", ip,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            stdout, _ = await asyncio.wait_for(proc.communicate(), timeout=15)
            output = stdout.decode("utf-8", errors="replace")

            if proc.returncode != 0:
                return {
                    "rtt_min": None, "rtt_avg": None, "rtt_max": None,
                    "packet_loss": 100.0, "is_reachable": False,
                }

            # Parse ping output
            result: dict[str, Any] = {
                "is_reachable": True,
                "packet_loss": 0.0,
            }

            for line in output.splitlines():
                if "packet loss" in line:
                    for part in line.split(","):
                        part = part.strip()
                        if "%" in part:
                            try:
                                result["packet_loss"] = float(
                                    part.split("%")[0].split()[-1]
                                )
                            except (ValueError, IndexError):
                                pass
                if "min/avg/max" in line:
                    try:
                        stats = line.split("=")[1].strip().split("/")
                        result["rtt_min"] = float(stats[0])
                        result["rtt_avg"] = float(stats[1])
                        result["rtt_max"] = float(stats[2])
                    except (ValueError, IndexError):
                        pass

            return result

        except (asyncio.TimeoutError, OSError) as e:
            logger.warning("Ping failed for %s: %s", ip, e)
            return {
                "rtt_min": None, "rtt_avg": None, "rtt_max": None,
                "packet_loss": 100.0, "is_reachable": False,
            }

    async def poll_device(self, device: dict) -> None:
        """Full poll cycle for a single device."""
        device_id = device["id"]
        hostname = device["hostname"]
        ip = clean_ip(device["ip_address"])
        now = datetime.now(timezone.utc)

        logger.info("Polling device %s (%s)", hostname, ip)
        poll_start = time.monotonic()

        try:
            # 1. Ping
            if device.get("ping_enabled", True):
                ping_result = await self.ping_device(device)
                db.insert_ping_metrics([{
                    "time": now,
                    "device_id": device_id,
                    **ping_result,
                }])

                if not ping_result["is_reachable"]:
                    db.update_device_status(device_id, "down")
                    logger.warning("Device %s (%s) is unreachable", hostname, ip)
                    return

            # 2. System info
            sys_info = await self.poll_system_info(device)
            if sys_info:
                device.update(sys_info)
                db.update_device_info(device_id, sys_info)
            else:
                db.update_device_status(device_id, "down")
                logger.warning("No SNMP response from %s (%s)", hostname, ip)
                return

            # 3. Interfaces
            interfaces = await self.poll_interfaces(device)
            if interfaces:
                # Save interface info
                iface_db = []
                for iface in interfaces:
                    iface_db.append({
                        "if_index": iface["if_index"],
                        "if_descr": iface.get("if_descr"),
                        "if_alias": iface.get("if_alias"),
                        "if_type": iface.get("if_type"),
                        "if_speed": iface.get("if_speed"),
                        "if_hc_speed": iface.get("if_hc_speed"),
                        "if_admin_status": iface.get("if_admin_status"),
                        "if_oper_status": iface.get("if_oper_status"),
                        "if_phys_address": iface.get("if_phys_address"),
                    })
                db.upsert_interfaces(device_id, iface_db)

                # Calculate traffic metrics with delta
                traffic_metrics = []
                counter_key = f"{device_id}"

                for iface in interfaces:
                    idx = iface["if_index"]
                    # Prefer 64-bit counters
                    in_octets = iface.get("in_octets_64", iface.get("in_octets_32"))
                    out_octets = iface.get("out_octets_64", iface.get("out_octets_32"))

                    prev_key = f"{counter_key}:{idx}"
                    prev = _previous_counters.get(prev_key)

                    in_bps = None
                    out_bps = None
                    if prev and in_octets is not None and out_octets is not None:
                        elapsed = (now - prev["time"]).total_seconds()
                        in_bps = calc_delta_bps(in_octets, prev["in_octets"], elapsed)
                        out_bps = calc_delta_bps(out_octets, prev["out_octets"], elapsed)

                    # Store current counters for next poll
                    if in_octets is not None and out_octets is not None:
                        _previous_counters[prev_key] = {
                            "time": now,
                            "in_octets": in_octets,
                            "out_octets": out_octets,
                        }

                    traffic_metrics.append({
                        "time": now,
                        "device_id": device_id,
                        "if_index": idx,
                        "in_octets": in_octets,
                        "out_octets": out_octets,
                        "in_bps": in_bps,
                        "out_bps": out_bps,
                        "in_errors": iface.get("in_errors", 0),
                        "out_errors": iface.get("out_errors", 0),
                    })

                db.insert_traffic_metrics(traffic_metrics)

            # 4. Host resources (CPU/Memory)
            hr = await self.poll_host_resources(device)
            
            if device.get("vendor") == "MikroTik":
                hr["pppoe_online"] = device.get("_pppoe_count", 0)

            db.insert_system_metrics([{
                "time": now,
                "device_id": device_id,
                "cpu_percent": hr.get("cpu_percent"),
                "memory_percent": hr.get("memory_percent"),
                "memory_used": hr.get("memory_used"),
                "memory_total": hr.get("memory_total"),
                "uptime": sys_info.get("sys_uptime"),
                "pppoe_online": hr.get("pppoe_online"),
            }])
            
            # 5. BGP Peers
            await self.poll_bgp_peers(device, now)

            elapsed = time.monotonic() - poll_start
            logger.info(
                "Poll complete for %s (%s) in %.1fs — %d interfaces",
                hostname, ip, elapsed, len(interfaces),
            )

        except Exception:
            logger.exception("Error polling device %s (%s)", hostname, ip)
            db.update_device_status(device_id, "down")

    async def poll_bgp_peers(self, device: dict, now: datetime) -> None:
        """Poll BGP peers and routes."""
        vendor = device.get("vendor", "")
        device_id = device["id"]
        peers = {}
        
        try:
            # Standard BGP4-MIB
            # 1.3.6.1.2.1.15.3.1.x
            # 2 = state, 3 = admin_status, 7 = remote_addr, 9 = remote_as, 10 = in_updates, 11 = out_updates
            
            bgp_data = await self._snmp_walk(device, Bgp4OIDs.BGP_PEER_TABLE)
            if bgp_data:
                # Group by suffix (which is the peer IP in standard MIB)
                for oid_str, val in bgp_data:
                    parts = oid_str.replace(Bgp4OIDs.BGP_PEER_TABLE + ".", "").split(".")
                    col = parts[0]
                    idx = ".".join(parts[1:])
                    
                    if idx not in peers:
                        peers[idx] = {"peer_addr": idx, "prefixes_received": None, "prefixes_advertised": None}
                        
                    if col == "2": peers[idx]["state"] = safe_int(val)
                    elif col == "3": peers[idx]["admin_status"] = safe_int(val)
                    elif col == "9": peers[idx]["peer_as"] = safe_int(val)
                    elif col == "10": peers[idx]["in_updates"] = safe_int(val)
                    elif col == "11": peers[idx]["out_updates"] = safe_int(val)

            # Huawei specific routes
            if vendor == "Huawei":
                hw_bgp = await self._snmp_walk(device, HuaweiOIDs.HW_BGP_PEER_ROUTES)
                if hw_bgp:
                    for oid_str, val in hw_bgp:
                        parts = oid_str.replace(HuaweiOIDs.HW_BGP_PEER_ROUTES + ".", "").split(".")
                        col = parts[0]
                        idx = ".".join(parts[4:]) # Skip address family indices? Wait, we can match by IP
                        # We just try to match the IP if it's in the oid
                        ip_parts = parts[-4:]
                        if len(ip_parts) == 4:
                            peer_ip = ".".join(ip_parts)
                            if peer_ip in peers:
                                if col == "1": # hwBgpPeerPrefixRecvCount
                                    # Since there can be multiple address families, we sum them
                                    val_int = safe_int(val)
                                    if val_int is not None:
                                        curr = peers[peer_ip].get("prefixes_received") or 0
                                        peers[peer_ip]["prefixes_received"] = curr + val_int

            if peers:
                db_peers = []
                for p in peers.values():
                    if p.get("peer_addr"):
                        db_peers.append({
                            "device_id": device_id,
                            "peer_addr": p.get("peer_addr"),
                            "peer_as": p.get("peer_as"),
                            "state": p.get("state"),
                            "admin_status": p.get("admin_status"),
                            "in_updates": p.get("in_updates"),
                            "out_updates": p.get("out_updates"),
                            "prefixes_received": p.get("prefixes_received"),
                            "prefixes_advertised": p.get("prefixes_advertised"),
                            "updated_at": now
                        })
                db.upsert_bgp_peers(device_id, db_peers)
                logger.info("Updated %d BGP peers for %s", len(db_peers), device.get("hostname"))

        except Exception as e:
            logger.warning("Error polling BGP for %s: %s", device.get("hostname"), e)

    async def _poll_loop(self, device: dict) -> None:
        """Run poll loop for a single device with its configured interval."""
        interval = device.get("poll_interval", 300)
        device_id = device["id"]
        hostname = device["hostname"]

        while self._running:
            await self.poll_device(device)
            logger.debug(
                "Next poll for %s in %ds", hostname, interval
            )
            await asyncio.sleep(interval)

    async def run(self) -> None:
        """Main poller loop — polls all enabled devices concurrently."""
        logger.info("SNMP Poller starting...")

        while self._running:
            devices = db.get_enabled_devices()
            if not devices:
                logger.info("No enabled devices found. Waiting 30s...")
                await asyncio.sleep(30)
                continue

            logger.info("Starting poll cycle for %d devices", len(devices))

            # Poll all devices concurrently
            tasks = [self.poll_device(d) for d in devices]
            await asyncio.gather(*tasks, return_exceptions=True)

            # Wait for the shortest interval before next cycle
            min_interval = min(d.get("poll_interval", 300) for d in devices)
            logger.info(
                "Poll cycle complete. Next cycle in %ds", min_interval
            )
            await asyncio.sleep(min_interval)
