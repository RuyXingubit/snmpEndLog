import pytest
import sys
import os

# Add parent directory to path so we can import snmp
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))

from snmp.oids import (
    SystemOIDs,
    InterfaceOIDs,
    HostResourcesOIDs,
    MikroTikOIDs,
    HuaweiOIDs,
    Bgp4OIDs,
    INTERFACE_HC_WALK_OIDS,
    detect_vendor
)

def test_oids_exist():
    # Test that common OIDs are correctly formed strings
    assert isinstance(SystemOIDs.SYS_DESCR, str)
    assert isinstance(HuaweiOIDs.HW_CPU_USAGE, str)
    assert isinstance(HuaweiOIDs.HW_MEMORY_USAGE, str)
    assert isinstance(HuaweiOIDs.HW_ENTITY_VOLTAGE, str)
    assert isinstance(MikroTikOIDs.FIRMWARE_VERSION, str)
    
def test_poller_imports_valid_oids():
    # Poller should import without throwing AttributeError for missing OIDs
    try:
        from snmp.poller import SNMPPoller
        poller = SNMPPoller()
        assert poller is not None
    except AttributeError as e:
        pytest.fail(f"Poller threw AttributeError on import due to bad OID: {e}")
