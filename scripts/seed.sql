-- Development seed. Run with: make seed
-- The statement is idempotent: existing site_ids are updated.
INSERT INTO stations (
    site_id, technology, end_id, contract_area_type, area_holder,
    infrastructure_type, ev_type, address, neighborhood, city, state,
    zip_code, regional, latitude, longitude, status, tower_type, nominal_aev,
    station_id, ev_supplier, land_area, structure_height, observation
) VALUES
    (
        'TSHG01', 'GSM', 'GOTSH_0001', 'Compartilhado', 'AMERICAN TOWER',
        'Greenfield', 'TORRE METALICA TRIANGULAR',
        'FAZENDA MUNDO NOVO, S/N, ZONA RURAL, TRES RANCHOS', 'ZONA RURAL',
        'TRES RANCHOS', 'GO', '75720000', 'TCO', -18.345528, -47.772028,
        'Aquisitado', '', '', 'TRH001TM', 'BRASILSAT',
        10.00, 50.00, 'VENDIDO GENESIS - 1º CLOSING'
    ),
    (
        'SPO01', '5G', 'GOSPO_0001', 'Compartilhado', 'AMERICAN TOWER',
        'Greenfield', 'TORRE METALICA QUADRADA',
        'AV PAULISTA, 1000', 'BELA VISTA', 'SAO PAULO', 'SP', '01310100',
        'SPO', -23.561684, -46.655981, 'Ativo', 'TORRE METALICA QUADRADA',
        '5G', 'TRHSP01', 'BRASILSAT', 12.50, 45.00, 'SITE URBANO - 5G'
    ),
    (
        'RJO02', 'LTE', 'GORJO_0002', 'Proprio', 'VIVO',
        'Rooftop', 'TORRE METALICA QUADRADA',
        'RUA DAS LARANJEIRAS, 100', 'BOTAFOGO', 'RIO DE JANEIRO', 'RJ',
        '22240000', 'RJO', -22.951900, -43.187500, 'Ativo', '', '',
        'TRHRJ02', 'ERICSSON', 8.00, 30.00, 'COBERTURA LTE'
    ),
    (
        'BHZ03', '5G', 'GOBHZ_0003', 'Compartilhado', 'TORRE BRASIL',
        'Greenfield', 'TORRE METALICA TRIANGULAR',
        'AV AFONSO PENA, 2000', 'SAVASSI', 'BELO HORIZONTE', 'MG', '30130000',
        'BHZ', -19.924500, -43.935200, 'Em processo', 'TORRE METALICA TRIANGULAR',
        '5G', 'TRHBH03', 'HUAWEI', 15.00, 55.00, 'NOVO SITE 5G'
    ),
    (
        'CWB04', 'LTE', 'GOCWB_0004', 'Compartilhado', 'AMERICAN TOWER',
        'Greenfield', 'TORRE METALICA QUADRADA',
        'AV BATEL, 300', 'BATEL', 'CURITIBA', 'PR', '80420000',
        'CWB', -25.437200, -49.269900, 'Ativo', 'TORRE METALICA QUADRADA',
        '', 'TRHCW04', 'NOKIA', 10.00, 40.00, 'COBERTURA LTE'
    ),
    (
        'POA05', 'GSM', 'GOPOA_0005', 'Proprio', 'TIM',
        'Greenfield', 'TORRE METALICA TRIANGULAR',
        'AV IPIRANGA, 400', 'CENTRO', 'PORTO ALEGRE', 'RS', '90010000',
        'POA', -30.034600, -51.217700, 'Aquisitado', '', '',
        'TRHPO05', 'BRASILSAT', 9.00, 35.00, 'LEGADO GSM'
    ),
    (
        'SSA06', '5G', 'GOSSA_0006', 'Compartilhado', 'TORRE BRASIL',
        'Greenfield', 'TORRE METALICA TRIANGULAR',
        'AV TANCREDO NEVES, 500', 'CAMINHO DAS ARVORES', 'SALVADOR', 'BA',
        '41820000', 'SSA', -12.977700, -38.501600, 'Ativo',
        'TORRE METALICA TRIANGULAR', '5G', 'TRHSS06', 'HUAWEI',
        14.00, 50.00, 'SITE 5G'
    ),
    (
        'REC07', 'LTE', 'GOREC_0007', 'Compartilhado', 'AMERICAN TOWER',
        'Greenfield', 'TORRE METALICA QUADRADA',
        'AV BOA VIAGEM, 600', 'BOA VIAGEM', 'RECIFE', 'PE', '51020000',
        'REC', -8.047600, -34.877000, 'Ativo', 'TORRE METALICA QUADRADA',
        '', 'TRHRE07', 'ERICSSON', 11.00, 42.00, 'COBERTURA LTE'
    ),
    (
        'FOR08', 'GSM', 'GOFOR_0008', 'Proprio', 'CLARO',
        'Greenfield', 'TORRE METALICA TRIANGULAR',
        'AV BEIRA MAR, 700', 'MEIRELES', 'FORTALEZA', 'CE', '60165000',
        'FOR', -3.731900, -38.526700, 'Aquisitado', '', '',
        'TRHFO08', 'BRASILSAT', 9.50, 38.00, 'LEGADO GSM'
    ),
    (
        'BSB09', '5G', 'GOBSB_0009', 'Compartilhado', 'TORRE BRASIL',
        'Greenfield', 'TORRE METALICA TRIANGULAR',
        'SCS QUADRA 2, BLOCO C', 'ASA SUL', 'BRASILIA', 'DF', '70300000',
        'BSB', -15.793900, -47.882800, 'Ativo', 'TORRE METALICA TRIANGULAR',
        '5G', 'TRHBS09', 'NOKIA', 13.00, 48.00, 'SITE 5G'
    ),
    (
        'MAO10', 'LTE', 'GOMAO_0010', 'Compartilhado', 'AMERICAN TOWER',
        'Greenfield', 'TORRE METALICA QUADRADA',
        'AV DJALMA BATISTA, 800', 'CHAPADA', 'MANAUS', 'AM', '69050000',
        'MAO', -3.119000, -60.021700, 'Em processo', 'TORRE METALICA QUADRADA',
        '', 'TRHMA10', 'HUAWEI', 12.00, 45.00, 'EXPANSAO LTE'
    ),
    (
        'FLN11', '5G', 'GOFLN_0011', 'Compartilhado', 'TORRE BRASIL',
        'Greenfield', 'TORRE METALICA TRIANGULAR',
        'AV BEIRA MAR NORTE, 900', 'CENTRO', 'FLORIANOPOLIS', 'SC', '88015000',
        'FLN', -27.595400, -48.548000, 'Ativo', 'TORRE METALICA TRIANGULAR',
        '5G', 'TRHFL11', 'ERICSSON', 10.50, 46.00, 'SITE 5G'
    ),
    (
        'NAT12', 'LTE', 'GONAT_0012', 'Proprio', 'VIVO',
        'Rooftop', 'TORRE METALICA QUADRADA',
        'AV ROBERTO FREIRE, 1000', 'PARNAMIRIM', 'NATAL', 'RN', '59140000',
        'NAT', -5.794500, -35.211000, 'Aquisitado', 'TORRE METALICA QUADRADA',
        '', 'TRHNA12', 'NOKIA', 7.00, 28.00, 'ROOFTOP LTE'
    ),
    (
        'CUI13', 'GSM', 'GOCUI_0013', 'Compartilhado', 'AMERICAN TOWER',
        'Greenfield', 'TORRE METALICA TRIANGULAR',
        'AV HISTORIADOR RUBENS DE MENDONCA, 1100', 'SANTA ROSA',
        'CUIABA', 'MT', '78040000', 'CUI', -15.601400, -56.097900,
        'Aquisitado', 'TORRE METALICA TRIANGULAR', '', 'TRHCU13', 'BRASILSAT',
        9.00, 36.00, 'LEGADO GSM'
    ),
    (
        'PEND14', 'LTE', 'GOPEND_0014', 'Compartilhado', 'AMERICAN TOWER',
        'Greenfield', 'TORRE METALICA QUADRADA',
        'RODOVIA BR-060, KM 10', 'ZONA RURAL', 'ANAPOLIS', 'GO', '75000000',
        'TCO', NULL, NULL, 'Em processo', 'TORRE METALICA QUADRADA', '',
        'TRHPE14', 'HUAWEI', NULL, NULL, 'COORDENADAS PENDENTES'
    )
ON DUPLICATE KEY UPDATE
    technology = VALUES(technology),
    end_id = VALUES(end_id),
    contract_area_type = VALUES(contract_area_type),
    area_holder = VALUES(area_holder),
    infrastructure_type = VALUES(infrastructure_type),
    ev_type = VALUES(ev_type),
    address = VALUES(address),
    neighborhood = VALUES(neighborhood),
    city = VALUES(city),
    state = VALUES(state),
    zip_code = VALUES(zip_code),
    regional = VALUES(regional),
    latitude = VALUES(latitude),
    longitude = VALUES(longitude),
    status = VALUES(status),
    tower_type = VALUES(tower_type),
    nominal_aev = VALUES(nominal_aev),
    station_id = VALUES(station_id),
    ev_supplier = VALUES(ev_supplier),
    land_area = VALUES(land_area),
    structure_height = VALUES(structure_height),
    observation = VALUES(observation);
