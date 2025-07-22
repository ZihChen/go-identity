-- Modify "players" table
ALTER TABLE `players` ADD COLUMN `level_id` bigint unsigned NOT NULL DEFAULT 0 AFTER `global_player_id`,
    ADD INDEX `idx_players_level_id` (`level_id`);
