-- ============================================
-- Migration 002: Vendor Detection
-- Adds columns for automatic vendor identification
-- and MikroTik-specific device information.
-- ============================================

ALTER TABLE devices ADD COLUMN IF NOT EXISTS vendor VARCHAR(64);
ALTER TABLE devices ADD COLUMN IF NOT EXISTS sys_object_id TEXT;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS board_name VARCHAR(255);
ALTER TABLE devices ADD COLUMN IF NOT EXISTS serial_number VARCHAR(255);
ALTER TABLE devices ADD COLUMN IF NOT EXISTS firmware_version VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_devices_vendor ON devices (vendor);
