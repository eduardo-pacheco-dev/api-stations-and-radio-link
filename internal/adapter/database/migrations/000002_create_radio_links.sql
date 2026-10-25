CREATE TABLE IF NOT EXISTS radio_links (
    link_id VARCHAR(50) NOT NULL,
    name VARCHAR(150) NOT NULL DEFAULT '',
    station_a_site_id VARCHAR(50) NOT NULL,
    station_b_site_id VARCHAR(50) NOT NULL,
    technology VARCHAR(50) NOT NULL DEFAULT '',
    frequency_mhz DECIMAL(10, 3) NULL,
    capacity_mbps DECIMAL(10, 2) NULL,
    distance_km DECIMAL(10, 3) NULL,
    status VARCHAR(50) NOT NULL DEFAULT '',
    observation TEXT NOT NULL,
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (link_id),
    KEY idx_radio_links_station_a (station_a_site_id),
    KEY idx_radio_links_station_b (station_b_site_id),
    KEY idx_radio_links_status (status),
    CONSTRAINT fk_radio_links_station_a
        FOREIGN KEY (station_a_site_id) REFERENCES stations (site_id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT fk_radio_links_station_b
        FOREIGN KEY (station_b_site_id) REFERENCES stations (site_id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT chk_radio_links_distinct_stations
        CHECK (station_a_site_id <> station_b_site_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
