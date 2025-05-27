-- 初始化資料庫結構

-- 商戶表
CREATE TABLE merchants (
    id BIGINT NOT NULL AUTO_INCREMENT,
    global_merchant_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    api_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_global_merchant_id (global_merchant_id),
    UNIQUE KEY uk_name (name),
    UNIQUE KEY uk_api_key (api_key)
);

-- 玩家表
CREATE TABLE players (
    id BIGINT NOT NULL AUTO_INCREMENT,
    merchant_id BIGINT NOT NULL,
    global_player_id VARCHAR(100) NOT NULL,
    api_key VARCHAR(255) NOT NULL,
    account VARCHAR(255) NOT NULL,
    email VARCHAR(255) NULL,
    last_active_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_global_player_id (global_player_id),
    UNIQUE KEY uk_api_key (api_key),
    KEY idx_merchant_account (merchant_id, account),
    CONSTRAINT fk_players_merchant FOREIGN KEY (merchant_id) REFERENCES merchants (id)
);

-- 管理員表
CREATE TABLE managers (
    id BIGINT NOT NULL AUTO_INCREMENT,
    merchant_id BIGINT NOT NULL,
    global_manager_id VARCHAR(100) NOT NULL,
    account VARCHAR(255) NOT NULL,
    email VARCHAR(255) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_global_manager_id (global_manager_id),
    KEY idx_merchant_account (merchant_id, account),
    CONSTRAINT fk_managers_merchant FOREIGN KEY (merchant_id) REFERENCES merchants (id)
);