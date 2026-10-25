-- Development seed for radio links. Run with: make seed
-- Applied after scripts/seed.sql, so the referenced stations exist.
INSERT INTO radio_links (
    link_id, name, station_a_site_id, station_b_site_id, technology,
    frequency_mhz, capacity_mbps, distance_km, status, observation
) VALUES
    (
        'RL-0001', 'TSHG01 <> CUI13', 'TSHG01', 'CUI13', 'MW',
        18000.000, 1000.00, 420.500, 'Ativo', 'LINK PRINCIPAL'
    ),
    (
        'RL-0002', 'SPO01 <> RJO02', 'SPO01', 'RJO02', 'MW',
        15000.000, 800.00, 360.200, 'Ativo', 'BACKBONE SP-RJ'
    ),
    (
        'RL-0003', 'BHZ03 <> BSB09', 'BHZ03', 'BSB09', 'MW',
        11000.000, 500.00, 620.800, 'Em processo', 'EXPANSAO MG-DF'
    ),
    (
        'RL-0004', 'CWB04 <> FLN11', 'CWB04', 'FLN11', 'MW',
        23000.000, 1000.00, 250.400, 'Ativo', 'BACKBONE PR-SC'
    ),
    (
        'RL-0005', 'SSA06 <> REC07', 'SSA06', 'REC07', 'MW',
        18000.000, 700.00, 680.100, 'Ativo', 'BACKBONE BA-PE'
    ),
    (
        'RL-0006', 'FOR08 <> NAT12', 'FOR08', 'NAT12', 'MW',
        15000.000, 600.00, 430.600, 'Aquisitado', 'ENLACE CE-RN'
    ),
    (
        'RL-0007', 'POA05 <> CWB04', 'POA05', 'CWB04', 'MW',
        11000.000, 500.00, 540.300, 'Ativo', 'BACKBONE RS-PR'
    )
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    station_a_site_id = VALUES(station_a_site_id),
    station_b_site_id = VALUES(station_b_site_id),
    technology = VALUES(technology),
    frequency_mhz = VALUES(frequency_mhz),
    capacity_mbps = VALUES(capacity_mbps),
    distance_km = VALUES(distance_km),
    status = VALUES(status),
    observation = VALUES(observation);
