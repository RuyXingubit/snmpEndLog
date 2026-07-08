import asyncio
import sys
import logging
from pysnmp.hlapi.asyncio import *

# logging.basicConfig(level=logging.DEBUG)

async def test_bgp(ip, community_str):
    print(f"Testing {ip} with {community_str}")
    community = CommunityData(community_str)
    transport = UdpTransportTarget((ip, 161), timeout=2.0, retries=1)
    
    # Check BGP peer states (1.3.6.1.2.1.15.3.1.2)
    iterator = walkCmd(
        SnmpEngine(),
        community,
        transport,
        ContextData(),
        ObjectType(ObjectIdentity('1.3.6.1.2.1.15.3.1.2')),
        lexicographicMode=False
    )
    
    found = False
    async for errorIndication, errorStatus, errorIndex, varBinds in iterator:
        if errorIndication:
            print(f"Error: {errorIndication}")
            break
        elif errorStatus:
            print(f"Status Error: {errorStatus}")
            break
        else:
            for varBind in varBinds:
                found = True
                print(f"BGP Peer: {varBind[0]} = {varBind[1]}")
    if not found:
        print("No standard BGP peers found.")
        
if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: test_bgp.py <ip> <community>")
        sys.exit(1)
    asyncio.run(test_bgp(sys.argv[1], sys.argv[2]))
