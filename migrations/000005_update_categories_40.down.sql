-- Migration: 000005_update_categories_40.down.sql
-- Rollback 40 categories to legacy set

-- Reset active state
UPDATE categories SET is_active = false WHERE id LIKE 'c0000000-%';
