"""nms Collector — SNMP Poller + Syslog Receiver.

Entry point that starts both services concurrently using asyncio.
"""

import asyncio
import logging
import os
import signal
import sys

import db
from config import Config
from db_migrate import run_migrations
from logs.receiver import SyslogReceiver
from snmp.poller import SNMPPoller
from snmp.ping_poller import PingPoller

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    stream=sys.stdout,
)
logger = logging.getLogger("nms")

RETENTION_DAYS = 90
CLEANUP_INTERVAL = 86400  # 24 hours in seconds


async def cleanup_scheduler() -> None:
    """Run data retention cleanup once per day.

    Deletes logs and metrics older than RETENTION_DAYS.
    First run after 60 seconds (let services stabilize), then every 24 hours.
    """
    await asyncio.sleep(60)  # Initial delay

    while True:
        try:
            loop = asyncio.get_event_loop()

            deleted_logs = await loop.run_in_executor(
                None, db.cleanup_old_logs, RETENTION_DAYS
            )
            deleted_metrics = await loop.run_in_executor(
                None, db.cleanup_old_metrics, RETENTION_DAYS
            )

            if deleted_logs > 0 or deleted_metrics > 0:
                logger.info(
                    "Data retention cleanup: deleted %d logs, %d metrics (older than %d days)",
                    deleted_logs, deleted_metrics, RETENTION_DAYS,
                )
            else:
                logger.debug("Data retention cleanup: nothing to delete")

        except Exception:
            logger.exception("Error during data retention cleanup")

        await asyncio.sleep(CLEANUP_INTERVAL)


async def main() -> None:
    """Start the collector services."""
    logger.info("=" * 60)
    logger.info("nms Collector starting...")
    logger.info("=" * 60)
    logger.info("DB: %s@%s:%d/%s", Config.DB_USER, Config.DB_HOST, Config.DB_PORT, Config.DB_NAME)
    logger.info("SNMP default interval: %ds", Config.SNMP_DEFAULT_INTERVAL)
    logger.info("Syslog UDP port: %d", Config.LOG_UDP_PORT)
    logger.info("Syslog TCP port: %d", Config.LOG_TCP_PORT)
    logger.info("Data retention: %d days", RETENTION_DAYS)

    # Run database migrations before anything else
    migrations_dir = os.environ.get("MIGRATIONS_DIR", "/app/db/migrations")
    run_migrations(Config.dsn(), migrations_dir)

    # Initialize database pool
    db.init_pool()

    # Create services
    poller = SNMPPoller()
    ping_poller = PingPoller()
    syslog = SyslogReceiver(
        udp_port=Config.LOG_UDP_PORT,
        tcp_port=Config.LOG_TCP_PORT,
    )

    # Handle graceful shutdown
    loop = asyncio.get_event_loop()
    shutdown_event = asyncio.Event()

    def _shutdown(sig: signal.Signals) -> None:
        logger.info("Received signal %s, shutting down...", sig.name)
        poller.stop()
        ping_poller.stop()
        syslog.stop()
        shutdown_event.set()

    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, _shutdown, sig)

    # Run services concurrently
    try:
        await asyncio.gather(
            poller.run(),
            ping_poller.run(),
            syslog.run(),
            cleanup_scheduler(),
        )
    except asyncio.CancelledError:
        logger.info("Services cancelled")
    finally:
        db.close_pool()
        logger.info("Collector stopped")


if __name__ == "__main__":
    asyncio.run(main())

