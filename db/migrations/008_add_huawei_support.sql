-- ============================================
-- Migration 008: Add Huawei Support (Temperature + VLAN L2)
-- ============================================

-- Add temperature to metric_system hypertable
ALTER TABLE metric_system ADD COLUMN IF NOT EXISTS temperature DOUBLE PRECISION;

-- Add L2 VLAN information to interfaces table
ALTER TABLE interfaces ADD COLUMN IF NOT EXISTS vlan_type VARCHAR(32);
ALTER TABLE interfaces ADD COLUMN IF NOT EXISTS native_vlan INTEGER;
