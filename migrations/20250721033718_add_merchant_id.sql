-- Modify "level" table
ALTER TABLE `level` ADD COLUMN `merchant_id` bigint unsigned NOT NULL, ADD INDEX `idx_level_merchant_id` (`merchant_id`);
-- Modify "tags" table
ALTER TABLE `tags` ADD COLUMN `merchant_id` bigint unsigned NOT NULL, ADD INDEX `idx_tags_merchant_id` (`merchant_id`);
