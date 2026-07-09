-- ============================================
-- nms — PostgreSQL Entrypoint
-- Only enables required extensions.
-- Schema is managed by migrations (db/migrations/).
-- ============================================

CREATE EXTENSION IF NOT EXISTS timescaledb;
