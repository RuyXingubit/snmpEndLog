"""Standard SNMP OIDs organized by category."""


# ============================================
# System MIB (RFC 1213 / SNMPv2-MIB)
# ============================================
class SystemOIDs:
    SYS_DESCR    = "1.3.6.1.2.1.1.1.0"
    SYS_OBJECT   = "1.3.6.1.2.1.1.2.0"
    SYS_UPTIME   = "1.3.6.1.2.1.1.3.0"
    SYS_CONTACT  = "1.3.6.1.2.1.1.4.0"
    SYS_NAME     = "1.3.6.1.2.1.1.5.0"
    SYS_LOCATION = "1.3.6.1.2.1.1.6.0"

# ============================================
# BGP4-MIB (RFC 4273)
# ============================================
class Bgp4OIDs:
    BGP_PEER_TABLE       = "1.3.6.1.2.1.15.3.1"
    PEER_STATE           = "1.3.6.1.2.1.15.3.1.2"
    PEER_ADMIN_STATUS    = "1.3.6.1.2.1.15.3.1.3"
    PEER_REMOTE_ADDR     = "1.3.6.1.2.1.15.3.1.7"
    PEER_REMOTE_AS       = "1.3.6.1.2.1.15.3.1.9"
    PEER_IN_UPDATES      = "1.3.6.1.2.1.15.3.1.10"
    PEER_OUT_UPDATES     = "1.3.6.1.2.1.15.3.1.11"
    PEER_FSM_ESTABLISHED_TIME = "1.3.6.1.2.1.15.3.1.16"



# ============================================
# Interface MIB (IF-MIB)
# ============================================
class InterfaceOIDs:
    # ifTable (32-bit counters)
    IF_NUMBER       = "1.3.6.1.2.1.2.1.0"
    IF_INDEX        = "1.3.6.1.2.1.2.2.1.1"
    IF_DESCR        = "1.3.6.1.2.1.2.2.1.2"
    IF_TYPE         = "1.3.6.1.2.1.2.2.1.3"
    IF_SPEED        = "1.3.6.1.2.1.2.2.1.5"
    IF_PHYS_ADDR    = "1.3.6.1.2.1.2.2.1.6"
    IF_ADMIN_STATUS = "1.3.6.1.2.1.2.2.1.7"
    IF_OPER_STATUS  = "1.3.6.1.2.1.2.2.1.8"
    IF_IN_OCTETS    = "1.3.6.1.2.1.2.2.1.10"
    IF_IN_ERRORS    = "1.3.6.1.2.1.2.2.1.14"
    IF_OUT_OCTETS   = "1.3.6.1.2.1.2.2.1.16"
    IF_OUT_ERRORS   = "1.3.6.1.2.1.2.2.1.20"

    # ifXTable (64-bit counters, preferred)
    IF_NAME          = "1.3.6.1.2.1.31.1.1.1.1"
    IF_HC_IN_OCTETS  = "1.3.6.1.2.1.31.1.1.1.6"
    IF_HC_OUT_OCTETS = "1.3.6.1.2.1.31.1.1.1.10"
    IF_HIGH_SPEED    = "1.3.6.1.2.1.31.1.1.1.15"
    IF_ALIAS         = "1.3.6.1.2.1.31.1.1.1.18"


# ============================================
# Host Resources MIB (HOST-RESOURCES-MIB)
# ============================================
class HostResourcesOIDs:
    HR_PROCESSOR_LOAD   = "1.3.6.1.2.1.25.3.3.1.2"    # CPU per core
    HR_STORAGE_DESCR    = "1.3.6.1.2.1.25.2.3.1.3"
    HR_STORAGE_ALLOC    = "1.3.6.1.2.1.25.2.3.1.4"     # allocation unit size
    HR_STORAGE_SIZE     = "1.3.6.1.2.1.25.2.3.1.5"
    HR_STORAGE_USED     = "1.3.6.1.2.1.25.2.3.1.6"
    HR_STORAGE_TYPE     = "1.3.6.1.2.1.25.2.3.1.2"

    # Storage type OIDs for filtering
    HR_STORAGE_RAM      = "1.3.6.1.2.1.25.2.1.2"


# ============================================
# OID walk targets (table prefixes for SNMP walk)
# ============================================
INTERFACE_WALK_OIDS = [
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
]

INTERFACE_HC_WALK_OIDS = [
    InterfaceOIDs.IF_NAME,
    InterfaceOIDs.IF_HC_IN_OCTETS,
    InterfaceOIDs.IF_HC_OUT_OCTETS,
    InterfaceOIDs.IF_HIGH_SPEED,
    InterfaceOIDs.IF_ALIAS,
]


# ============================================
# MikroTik (enterprise 14988)
# ============================================
class MikroTikOIDs:
    # mtxrHealth  = 1.3.6.1.4.1.14988.1.1.3
    HL_VOLTAGE          = "1.3.6.1.4.1.14988.1.1.3.8.0"    # input voltage (dV)
    HL_TEMPERATURE      = "1.3.6.1.4.1.14988.1.1.3.10.0"   # temperature (dC)
    HL_CPU_TEMPERATURE  = "1.3.6.1.4.1.14988.1.1.3.6.0"    # CPU temp (dC)
    HL_BOARD_TEMP       = "1.3.6.1.4.1.14988.1.1.3.7.0"    # board temp (dC)
    HL_POWER            = "1.3.6.1.4.1.14988.1.1.3.12.0"   # power (dW)
    HL_CURRENT          = "1.3.6.1.4.1.14988.1.1.3.13.0"   # current (mA)
    HL_CPU_FREQ         = "1.3.6.1.4.1.14988.1.1.3.14.0"   # CPU freq (MHz)

    # mtxrSystem = 1.3.6.1.4.1.14988.1.1.7
    SERIAL_NUMBER       = "1.3.6.1.4.1.14988.1.1.7.3.0"
    FIRMWARE_VERSION    = "1.3.6.1.4.1.14988.1.1.7.4.0"
    BOARD_NAME          = "1.3.6.1.4.1.14988.1.1.7.9.0"

    # mtxrLicense = 1.3.6.1.4.1.14988.1.1.4
    LICENSE_LEVEL       = "1.3.6.1.4.1.14988.1.1.4.4.0"

    # Enterprise OID prefix
    ENTERPRISE_PREFIX   = "1.3.6.1.4.1.14988"


# ============================================
# Huawei (enterprise 2011)
# ============================================
class HuaweiOIDs:
    # HUAWEI-DEVICE-EXT-MIB
    SERIAL_NUMBER           = "1.3.6.1.4.1.2011.5.25.188.1.1.0"
    
    # HUAWEI-ENTITY-EXTENT-MIB (1.3.6.1.4.1.2011.5.25.31)
    HW_CPU_USAGE        = "1.3.6.1.4.1.2011.5.25.31.1.1.1.1.5"
    HW_MEMORY_USAGE     = "1.3.6.1.4.1.2011.5.25.31.1.1.1.1.7"
    HW_TEMPERATURE      = "1.3.6.1.4.1.2011.5.25.31.1.1.1.1.11"
    HW_ENTITY_VOLTAGE   = "1.3.6.1.4.1.2011.5.25.31.1.1.1.1.13"

    # HUAWEI-BRAS-PPPOE-MIB
    PPPOE_ONLINE        = "1.3.6.1.4.1.2011.5.25.106.2.1.2.1.1.2"
    
    # HUAWEI-BGP-VPN-MIB
    HW_BGP_PEER_TABLE   = "1.3.6.1.4.1.2011.5.25.177.1.1.2.1"
    HW_BGP_PEER_ROUTES  = "1.3.6.1.4.1.2011.5.25.177.1.1.3.1"

    # HUAWEI-L2IF-MIB (VLANs / Port types)
    HW_L2_PORT_TYPE     = "1.3.6.1.4.1.2011.5.25.42.1.1.1.3.1.3"
    HW_L2_PVID          = "1.3.6.1.4.1.2011.5.25.42.1.1.1.3.1.4"

    # Board name
    HW_BOARD_NAME           = "1.3.6.1.4.1.2011.6.157.2.1.1.3"
    
    # PPPoE Online Users
    HW_TOTAL_PPPOE_ONLINE_NUM = "1.3.6.1.4.1.2011.5.2.1.14.1.2.0"
    
    
    ENTERPRISE_PREFIX       = "1.3.6.1.4.1.2011"



# ============================================
# Vendor detection by enterprise OID prefix
# ============================================
VENDOR_MAP = {
    "1.3.6.1.4.1.14988": "MikroTik",
    "1.3.6.1.4.1.2636":  "Juniper",
    "1.3.6.1.4.1.9":     "Cisco",
    "1.3.6.1.4.1.2011":  "Huawei",
    "1.3.6.1.4.1.6486":  "Alcatel-Lucent",
    "1.3.6.1.4.1.890":   "ZyXEL",
    "1.3.6.1.4.1.25506":  "H3C",
    "1.3.6.1.4.1.4413":  "Ubiquiti",
    "1.3.6.1.4.1.8072":  "Net-SNMP (Linux)",
    "1.3.6.1.4.1.311":   "Microsoft",
}


def detect_vendor(sys_object_id: str) -> str:
    """Detect vendor from sysObjectID OID string."""
    if not sys_object_id:
        return "Desconhecido"
    for prefix, vendor in VENDOR_MAP.items():
        if sys_object_id.startswith(prefix):
            return vendor
    return "Desconhecido"
