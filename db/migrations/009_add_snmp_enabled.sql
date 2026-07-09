-- Add snmp_enabled column to devices table
ALTER TABLE devices ADD COLUMN snmp_enabled BOOLEAN DEFAULT TRUE;
