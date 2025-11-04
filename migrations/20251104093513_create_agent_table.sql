-- Create "agents" table
CREATE TABLE `agents` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `merchant_id` bigint unsigned NOT NULL,
  `global_agent_id` varchar(255) NOT NULL,
  `account` varchar(255) NOT NULL,
  `ancestry` text NULL,
  `current_sign_in_at` datetime NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_agents_account` (`account`),
  INDEX `idx_agents_current_sign_in_at` (`current_sign_in_at`),
  INDEX `idx_agents_deleted_at` (`deleted_at`),
  INDEX `idx_agents_merchant_id` (`merchant_id`),
  INDEX `idx_agents_updated_at` (`updated_at`),
  UNIQUE INDEX `unique_global_agent_id` (`global_agent_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
