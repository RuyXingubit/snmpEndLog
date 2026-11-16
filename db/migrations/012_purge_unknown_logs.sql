-- ============================================
-- Migration 012: Purge logs from unknown/unregistered devices
-- ============================================

DELETE FROM logs
WHERE device_id IS NULL
   OR NOT EXISTS (SELECT 1 FROM devices d WHERE d.id = logs.device_id);
