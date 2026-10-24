INSERT IGNORE INTO stations (
    site_id, technology, end_id, contract_area_type, area_holder,
    infrastructure_type, ev_type, address, neighborhood, city, state,
    zip_code, regional, latitude, longitude, status, tower_type, nominal_aev,
    station_id, ev_supplier, land_area, structure_height, observation
) VALUES (
    'TSHG01', 'GSM', 'GOTSH_0001', 'Compartilhado', 'AMERICAN TOWER',
    'Greenfield', 'TORRE METALICA TRIANGULAR',
    'FAZENDA MUNDO NOVO, S/N, ZONA RURAL, TRES RANCHOS', 'ZONA RURAL',
    'TRES RANCHOS', 'GO', '75720000', 'TCO', -18.345528, -47.772028,
    'Aquisitado', '', '', 'TRH001TM', 'BRASILSAT',
    10.00, 50.00, 'VENDIDO GENESIS - 1º CLOSING'
);
