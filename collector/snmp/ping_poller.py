"""Ping Poller for background latency and packet loss tracking."""

import asyncio
import logging
import time
from datetime import datetime, timezone
from typing import Any

import db

logger = logging.getLogger(__name__)


def clean_ip(ip: str) -> str:
    """Remove subnet mask from IP string if present."""
    return ip.split("/")[0].strip()


class PingPoller:
    """Manages continuous background ICMP polling for all devices."""

    def __init__(self):
        self._running = False
        self._tasks: dict[int, asyncio.Task] = {}

    async def _ping_loop(self, device: dict) -> None:
        """Continuous loop for a single device, doing 60 pings every 60 seconds."""
        ip = clean_ip(device["ip_address"])
        device_id = device["id"]

        while self._running:
            start_time = time.monotonic()
            now = datetime.now(timezone.utc)
            
            try:
                # -c 60: 60 packets
                # -i 1: 1 packet per second
                # -q: quiet output (only summary)
                # -w 65: timeout the whole command after 65 seconds
                proc = await asyncio.create_subprocess_exec(
                    "ping", "-c", "60", "-i", "1", "-w", "65", "-q", ip,
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.PIPE,
                )
                
                stdout, _ = await asyncio.wait_for(proc.communicate(), timeout=70)
                output = stdout.decode("utf-8", errors="replace")

                result: dict[str, Any] = {
                    "rtt_min": None, "rtt_avg": None, "rtt_max": None,
                    "packet_loss": 100.0, "is_reachable": False,
                }

                if proc.returncode == 0 or proc.returncode == 1:
                    # Parse ping summary
                    result["is_reachable"] = True
                    result["packet_loss"] = 0.0
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

                    if result["packet_loss"] == 100.0:
                        result["is_reachable"] = False

                db.insert_ping_metrics([{
                    "time": now,
                    "device_id": device_id,
                    **result,
                }])

                if not result["is_reachable"]:
                    db.update_device_status(device_id, "down")

            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.warning("Ping error for %s: %s", ip, e)

            # Ensure the loop cycles roughly every 60s even if ping fails instantly
            elapsed = time.monotonic() - start_time
            if elapsed < 60:
                await asyncio.sleep(60 - elapsed)

    async def run(self) -> None:
        """Main loop that spawns and manages device ping tasks."""
        logger.info("Ping Poller starting...")
        self._running = True

        while self._running:
            try:
                devices = db.get_enabled_devices()
                current_ids = set()

                for d in devices:
                    if not d.get("ping_enabled", True):
                        continue
                        
                    current_ids.add(d["id"])
                    if d["id"] not in self._tasks:
                        logger.info("Starting background ping loop for %s", d["ip_address"])
                        self._tasks[d["id"]] = asyncio.create_task(self._ping_loop(d))

                # Stop tracking removed or disabled devices
                for dev_id in list(self._tasks.keys()):
                    if dev_id not in current_ids:
                        logger.info("Stopping background ping loop for device %d", dev_id)
                        self._tasks[dev_id].cancel()
                        del self._tasks[dev_id]
                        
            except Exception as e:
                logger.error("PingPoller manager error: %s", e)

            # Check for new devices every 60 seconds
            await asyncio.sleep(60)

    def stop(self) -> None:
        """Stop all ping polling tasks."""
        self._running = False
        for t in self._tasks.values():
            t.cancel()
