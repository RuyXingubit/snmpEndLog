-- Migration: Add pppoe_online column to metric_system
ALTER TABLE metric_system ADD COLUMN IF NOT EXISTS pppoe_online INTEGER;
