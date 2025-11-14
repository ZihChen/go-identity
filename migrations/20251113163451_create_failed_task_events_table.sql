-- Create "failed_task_events" table
CREATE TABLE `failed_task_events` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `task_id` varchar(255) NOT NULL,
  `task_type` varchar(100) NOT NULL,
  `queue_name` varchar(100) NOT NULL DEFAULT 'default',
  `payload` longtext NOT NULL,
  `error_message` text NOT NULL,
  `retry_count` int NOT NULL DEFAULT 0,
  `failed_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `redis_key` varchar(500) NULL,
  `redis_state` varchar(50) NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `idx_failed_task_events_task_id` (`task_id`),
  INDEX `idx_failed_task_events_task_type` (`task_type`),
  INDEX `idx_failed_task_events_queue_name` (`queue_name`),
  INDEX `idx_failed_task_events_failed_at` (`failed_at`),
  INDEX `idx_failed_task_events_redis_state` (`redis_state`),
  INDEX `idx_failed_task_events_retry_count` (`retry_count`),
  INDEX `idx_failed_task_events_created_at` (`created_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;