-- Create "managers" table
CREATE TABLE `managers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `merchant_id` bigint unsigned NOT NULL,
  `global_manager_id` varchar(100) NOT NULL,
  `account` varchar(255) NOT NULL,
  `email` varchar(255) NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_managers_deleted_at` (`deleted_at`),
  UNIQUE INDEX `idx_managers_global_manager_id` (`global_manager_id`),
  INDEX `idx_managers_merchant_id` (`merchant_id`),
  INDEX `idx_merchant_account` (`account`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "merchants" table
CREATE TABLE `merchants` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `global_merchant_id` varchar(100) NOT NULL,
  `name` varchar(255) NOT NULL,
  `display_name` varchar(255) NOT NULL,
  `api_key` varchar(255) NOT NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_merchants_api_key` (`api_key`),
  INDEX `idx_merchants_deleted_at` (`deleted_at`),
  UNIQUE INDEX `idx_merchants_global_merchant_id` (`global_merchant_id`),
  UNIQUE INDEX `idx_merchants_name` (`name`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "players" table
CREATE TABLE `players` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `merchant_id` bigint unsigned NOT NULL,
  `global_player_id` varchar(100) NOT NULL,
  `api_key` varchar(255) NOT NULL,
  `account` varchar(255) NOT NULL,
  `email` varchar(255) NULL,
  `last_active_at` datetime NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_merchant_account` (`account`),
  UNIQUE INDEX `idx_players_api_key` (`api_key`),
  INDEX `idx_players_deleted_at` (`deleted_at`),
  UNIQUE INDEX `idx_players_global_player_id` (`global_player_id`),
  INDEX `idx_players_merchant_id` (`merchant_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
