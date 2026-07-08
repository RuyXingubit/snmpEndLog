"""Centralized configuration from environment variables."""

import os


class Config:
    """Application configuration loaded from environment."""

    # Database
    DB_HOST: str = os.getenv("DB_HOST", "localhost")
    DB_PORT: int = int(os.getenv("DB_PORT", "5432"))
    DB_NAME: str = os.getenv("DB_NAME", "snmpendlog")
    DB_USER: str = os.getenv("DB_USER", "snmpendlog")
    DB_PASSWORD: str = os.getenv("DB_PASSWORD", "")

    # SNMP
    SNMP_DEFAULT_INTERVAL: int = int(os.getenv("SNMP_DEFAULT_INTERVAL", "300"))

    # Syslog receiver
    LOG_UDP_PORT: int = int(os.getenv("LOG_UDP_PORT", "514"))
    LOG_TCP_PORT: int = int(os.getenv("LOG_TCP_PORT", "514"))

    @classmethod
    def dsn(cls) -> str:
        """Return PostgreSQL connection string."""
        return (
            f"host={cls.DB_HOST} port={cls.DB_PORT} dbname={cls.DB_NAME} "
            f"user={cls.DB_USER} password={cls.DB_PASSWORD}"
        )
