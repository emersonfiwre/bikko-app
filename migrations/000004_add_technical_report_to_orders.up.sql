-- Migration: Add technical_report (up to 280 chars) to orders table
ALTER TABLE orders ADD COLUMN IF NOT EXISTS technical_report VARCHAR(280);
