-- Creates the local development database.
-- Usage (when not using docker compose):
--   mysql -uroot -padmin < scripts/init.sql

CREATE DATABASE IF NOT EXISTS api_stations_radio_link
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;
