"""Database migration runner.

Applies SQL migrations in order, tracking applied versions
in a `schema_migrations` table.
"""

import logging
import os
import re

import psycopg2

logger = logging.getLogger("migrate")

MIGRATIONS_TABLE = "schema_migrations"


def _ensure_migrations_table(conn) -> None:
    """Create the schema_migrations table if it doesn't exist."""
    with conn.cursor() as cur:
        cur.execute(f"""
            CREATE TABLE IF NOT EXISTS {MIGRATIONS_TABLE} (
                version     INTEGER PRIMARY KEY,
                name        TEXT NOT NULL,
                applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
            );
        """)
    conn.commit()


def _get_applied_versions(conn) -> set[int]:
    """Return the set of already-applied migration version numbers."""
    with conn.cursor() as cur:
        cur.execute(f"SELECT version FROM {MIGRATIONS_TABLE}")
        return {row[0] for row in cur.fetchall()}


def _discover_migrations(migrations_dir: str) -> list[tuple[int, str, str]]:
    """Discover migration files and return sorted list of (version, name, filepath).

    Migration files must be named like: 001_description.sql
    """
    pattern = re.compile(r"^(\d{3})_(.+)\.sql$")
    migrations = []

    if not os.path.isdir(migrations_dir):
        logger.warning("Migrations directory not found: %s", migrations_dir)
        return migrations

    for filename in sorted(os.listdir(migrations_dir)):
        match = pattern.match(filename)
        if match:
            version = int(match.group(1))
            name = match.group(2)
            filepath = os.path.join(migrations_dir, filename)
            migrations.append((version, name, filepath))

    return migrations


def run_migrations(dsn: str, migrations_dir: str) -> None:
    """Connect to the database and apply any pending migrations.

    Args:
        dsn: PostgreSQL connection string.
        migrations_dir: Absolute path to directory containing .sql migration files.
    """
    logger.info("Checking database migrations...")

    conn = psycopg2.connect(dsn)
    conn.autocommit = False

    try:
        _ensure_migrations_table(conn)
        applied = _get_applied_versions(conn)
        migrations = _discover_migrations(migrations_dir)

        if not migrations:
            logger.warning("No migration files found in %s", migrations_dir)
            return

        pending = [(v, n, f) for v, n, f in migrations if v not in applied]

        if not pending:
            logger.info(
                "Database is up to date (version %d, %d migrations applied)",
                max(applied) if applied else 0,
                len(applied),
            )
            return

        for version, name, filepath in pending:
            logger.info("Applying migration %03d_%s ...", version, name)

            with open(filepath, "r") as f:
                sql = f.read()

            try:
                with conn.cursor() as cur:
                    cur.execute(sql)
                    cur.execute(
                        f"INSERT INTO {MIGRATIONS_TABLE} (version, name) VALUES (%s, %s)",
                        (version, name),
                    )
                conn.commit()
                logger.info("  ✓ Migration %03d_%s applied successfully", version, name)
            except Exception as e:
                conn.rollback()
                logger.error("  ✗ Migration %03d_%s FAILED: %s", version, name, e)
                raise RuntimeError(
                    f"Migration {version:03d}_{name} failed: {e}"
                ) from e

        logger.info(
            "All migrations applied. Database at version %d (%d total)",
            pending[-1][0],
            len(applied) + len(pending),
        )
    finally:
        conn.close()
