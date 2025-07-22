-- Create "player_tags" table
CREATE TABLE `player_tags` (
  `player_id` bigint unsigned NULL,
  `tag_id` bigint unsigned NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE INDEX `idx_player_tag` (`player_id`, `tag_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
