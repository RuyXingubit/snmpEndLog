-- Migration: Add voltage column to devices
ALTER TABLE devices ADD COLUMN voltage REAL;
