# API Stations and Radio Link

API Go para gerenciamento de estações e enlace de rádio. Nesta primeira versão
existe apenas o endpoint de status da API, com verificação de conectividade com
o banco de dados.

## Stack

- Go 1.24+
- `net/http` (sem framework web)
- MySQL 8.4
- GORM + driver MySQL
- Swagger/OpenAPI via `swaggo/swag` + `http-swagger`
- SQLite em memória para testes
- `log/slog` para logs estruturados
- PM2 em produção
- GitHub Actions para CI/CD

## Estrutura

```text
.
├── cmd/api/main.go                         # composition root, servidor e shutdown
├── docs/                                   # documentação Swagger gerada
├── internal/
│   ├── adapter/
│   │   ├── database/                       # abertura GORM, pool, migrações
│   │   ├── gormerr/                        # classificação de erros GORM
│   │   ├── gormmodel/                      # modelos GORM
│   │   ├── httpapi/                        # rotas, handlers, middleware
│   │   ├── importjobmysql/                 # repositório GORM de jobs
│   │   ├── importprocessor/                # executor de jobs em lotes
│   │   ├── radiolinkmysql/                 # repositório GORM de radio links
│   │   ├── spreadsheet/                    # parser de importação Excel
│   │   ├── sqlpinger/                      # adapter de ping no banco
│   │   └── stationmysql/                   # repositório GORM de stations
│   ├── config/                             # configuração via variáveis de ambiente
│   ├── importjob/                          # domínio, storage, fila e runner
│   ├── radiolink/                          # domínio, casos de uso e porta Repository
│   ├── station/                            # domínio, casos de uso e porta Repository
│   ├── status/                             # caso de uso + portas + read model
│   └── testutil/sqlitedb/                  # SQLite em memória para testes
├── scripts/
│   ├── init.sql                            # cria o banco local
│   ├── seed.sql                            # stations de exemplo
│   └── seed_radio_links.sql                # radio links de exemplo
├── docker-compose.yml                      # MySQL local (root/admin)
├── deploy.sh                               # deploy atômico com PM2, healthcheck e rollback
├── ecosystem.config.cjs                    # configuração do PM2
├── .github/workflows/                      # CI e deploy automático
├── Makefile
└── .env.example
```

### Regra de dependência

```text
HTTP/MySQL (adapters) ──> status (core) <── DatabasePinger (porta)
```

`internal/status` não importa HTTP, MySQL ou nenhuma biblioteca de infraestrutura.
O caso de uso depende apenas da interface `DatabasePinger`. O driver MySQL é
registrado somente em `cmd/api/main.go`.

## Skills instaladas

As skills foram instaladas em `.agents/skills/` (formato compatível com OpenCode)
via `npx skills add`:

| Skill | Origem | Tema |
| --- | --- | --- |
| `golang-code-style` | `samber/cc-skills-golang` | Go / clean code |
| `golang-code-quality` | `jkeddari/go-skills` | Go / clean code |
| `golang-testing` | `samber/cc-skills-golang` | Testes unitários e de integração |
| `golang-database` | `samber/cc-skills-golang` | Banco de dados |
| `golang-project-layout` | `samber/cc-skills-golang` | Layout de projeto Go |
| `golang-swagger` | `samber/cc-skills-golang` | Swagger / OpenAPI |
| `golang-architecture` | `jkeddari/go-skills` | Clean Architecture em Go |
| `clean-architecture` | `krzysztofsurdy/code-virtuoso` | Clean Architecture |
| `solid` | `krzysztofsurdy/code-virtuoso` | SOLID |
| `api-design` | `krzysztofsurdy/code-virtuoso` | API Design |
| `tdd` | `krzysztofsurdy/code-virtuoso` | TDD |

Para reinstalar/atualizar:

```bash
npx skills update
```

> As skills `api-design`, `solid` e `clean-architecture` tiveram a diretiva
> `disable-model-invocation` removida para que o OpenCode as descubra
> automaticamente. Um `npx skills update` pode restaurar essa diretiva; remova-a
> novamente se quiser que elas continuem sendo sugeridas pelo agente.

## Pré-requisitos

- Go 1.24+
- Docker (opcional, para o MySQL local)
- MySQL 8+ acessível (caso não use Docker)

## Como rodar

O ambiente de desenvolvimento usa o MySQL local:

| Configuração | Valor |
| --- | --- |
| Host | `127.0.0.1:3306` |
| Usuário | `root` |
| Senha | `admin` |
| Banco | `api_stations_radio_link` |

1. Crie o banco uma vez, caso ainda não exista:

```bash
mysql -uroot -padmin < scripts/init.sql
```

As tabelas são criadas automaticamente pelas migrações ao iniciar a API.

2. Crie o arquivo de ambiente:

```bash
cp .env.example .env
```

3. Rode a API:

```bash
go run ./cmd/api
```

Ou:

```bash
make run
```

### Alternativa com Docker

Se preferir subir o MySQL via Docker, o container usa as mesmas credenciais:

```bash
docker compose up -d mysql
```

### Hot reload com Air

O projeto inclui `.air.toml` apontando para `./cmd/api`. Na raiz do projeto:

```bash
air
```

O binário é gerado em `tmp/` (ignorado pelo Git) e o servidor reinicia ao
alterar arquivos `.go`.

A API sobe em `http://localhost:8080`. Se o MySQL estiver indisponível, a API
continua de pé e o endpoint de status responde `503` com estado `degraded`.

## Endpoint

### `GET /api/v1/status`

Resposta saudável (`200 OK`):

```json
{
  "service": "api-stations-and-radio-link",
  "version": "1.0.0",
  "environment": "development",
  "status": "ok",
  "database": {
    "status": "up",
    "driver": "mysql",
    "latency_ms": 1.42
  },
  "uptime": "1m30s",
  "timestamp": "2026-10-24T12:00:00Z"
}
```

Banco indisponível (`503 Service Unavailable`):

```json
{
  "service": "api-stations-and-radio-link",
  "version": "1.0.0",
  "environment": "development",
  "status": "degraded",
  "database": {
    "status": "down",
    "driver": "mysql",
    "error": "database unreachable"
  },
  "uptime": "1m30s",
  "timestamp": "2026-10-24T12:00:00Z"
}
```

### `GET /api/v1/healthz`

Liveness probe independente do banco:

```json
{"status": "ok"}
```

Retorna `200` sempre que o processo HTTP está no ar. Use este endpoint no
healthcheck de deploy; o `/api/v1/status` continua reportando a saúde do banco
e pode retornar `503`.

## Stations

CRUD em `/api/v1/stations`. O identificador do recurso é o `site_id`.

| Método | Rota | Descrição |
| --- | --- | --- |
| `POST` | `/api/v1/stations` | Cria uma estação |
| `GET` | `/api/v1/stations` | Lista com paginação e busca |
| `GET` | `/api/v1/stations/{siteID}` | Busca por `site_id` |
| `PUT` | `/api/v1/stations/{siteID}` | Substitui os dados da estação |
| `DELETE` | `/api/v1/stations/{siteID}` | Remove a estação |

A listagem aceita `page` (padrão `1`), `page_size` (padrão `20`, máximo `100`)
e `search` (busca em `site_id`, `end_id`, `station_id`, `city` e `address`).

O `PUT` substitui todos os campos mutáveis; campos omitidos são gravados como
vazios/nulos. O `site_id` é imutável e deve ser igual ao da rota.

### Exemplo de criação

```bash
curl -X POST http://localhost:8080/api/v1/stations \
  -H "Content-Type: application/json" \
  -d '{
    "site_id": "TSHG01",
    "technology": "GSM",
    "end_id": "GOTSH_0001",
    "contract_area_type": "Compartilhado",
    "area_holder": "AMERICAN TOWER",
    "infrastructure_type": "Greenfield",
    "ev_type": "TORRE METALICA TRIANGULAR",
    "address": "FAZENDA MUNDO NOVO, S/N, ZONA RURAL, TRES RANCHOS",
    "neighborhood": "ZONA RURAL",
    "city": "TRES RANCHOS",
    "state": "GO",
    "zip_code": "75720000",
    "regional": "TCO",
    "latitude": -18.345528,
    "longitude": -47.772028,
    "status": "Aquisitado",
    "tower_type": "",
    "nominal_aev": "",
    "station_id": "TRH001TM",
    "ev_supplier": "BRASILSAT",
    "land_area": 10,
    "structure_height": 50,
    "observation": "VENDIDO GENESIS - 1º CLOSING"
  }'
```

### Mapeamento dos campos

| Origem | API |
| --- | --- |
| `SITE ID` | `site_id` |
| `TECNOLOGIA` | `technology` |
| `END ID` | `end_id` |
| `TIPO_DE_CONTRATO_AREA` | `contract_area_type` |
| `DETENTOR_AREA` | `area_holder` |
| `TIPO_DE_INFRA` | `infrastructure_type` |
| `TIPO_DE_EV` | `ev_type` |
| `ENDERECO` | `address` |
| `BAIRRO` | `neighborhood` |
| `MUNICIPIO` | `city` |
| `ESTADO` | `state` |
| `CEP` | `zip_code` |
| `REGIONAL` | `regional` |
| `LATITUDE` | `latitude` |
| `LONGITUDE` | `longitude` |
| `STATUS` | `status` |
| `TIPO_DA_TORRE` | `tower_type` |
| `AEV_NOMINAL` | `nominal_aev` |
| `STATION_ID` | `station_id` |
| `FORNECEDOR_DE_EV` | `ev_supplier` |
| `AREA_DO_SOLO` | `land_area` |
| `ALTURA_DA_ESTRUTURA` | `structure_height` |
| `OBSERVACAO_THQ` | `observation` |

As coordenadas são números decimais na API: `-18,345528` vira `-18.345528`.

## Radio Links

CRUD em `/api/v1/radio-links`. Cada link possui uma ponta A e uma ponta B,
referenciando `stations.site_id`.

| Método | Rota | Descrição |
| --- | --- | --- |
| `POST` | `/api/v1/radio-links` | Cria um radio link |
| `GET` | `/api/v1/radio-links` | Lista com paginação, busca e filtro por estação |
| `GET` | `/api/v1/radio-links/{linkID}` | Busca por `link_id` |
| `PUT` | `/api/v1/radio-links/{linkID}` | Substitui os dados do link |
| `DELETE` | `/api/v1/radio-links/{linkID}` | Remove o link |

A listagem aceita `page`, `page_size`, `search` (`link_id`, `name`, estações e
tecnologia) e `station_id` (links em que a estação é ponta A ou B).

Regras:

- `station_a_site_id` e `station_b_site_id` precisam existir
- as duas pontas devem ser diferentes
- estação referenciada por um link não pode ser removida (`409 station_in_use`)
- `link_id` é imutável

### Exemplo de criação

```bash
curl -X POST http://localhost:8080/api/v1/radio-links \
  -H "Content-Type: application/json" \
  -d '{
    "link_id": "RL-0001",
    "name": "TSHG01 <> CUI13",
    "station_a_site_id": "TSHG01",
    "station_b_site_id": "CUI13",
    "technology": "MW",
    "frequency_mhz": 18000,
    "capacity_mbps": 1000,
    "distance_km": 420.5,
    "status": "Ativo",
    "observation": "LINK PRINCIPAL"
  }'
```

## Importação em massa (Excel)

Dois endpoints aceitam arquivos `.xlsx` (primeira aba):

| Método | Rota | Descrição |
| --- | --- | --- |
| `POST` | `/api/v1/stations/import` | Importa stations |
| `POST` | `/api/v1/radio-links/import` | Importa radio links |

Parâmetros:

- `file` (multipart/form-data): arquivo `.xlsx`, máximo 10 MiB
- `mode`: `insert` (padrão) ou `upsert`
- `dry_run`: `true` valida sem gravar

Exemplo:

```bash
curl -X POST "http://localhost:8080/api/v1/stations/import?mode=upsert" \
  -F "file=@stations.xlsx"
```

Resposta:

```json
{
  "total": 100,
  "valid": 95,
  "imported": 90,
  "updated": 5,
  "failed": 5,
  "dry_run": false,
  "errors": [
    {
      "row": 12,
      "site_id": "TSHG12",
      "message": "state must be a valid Brazilian UF"
    }
  ]
}
```

Linhas válidas são importadas mesmo quando outras falham. O número da linha
corresponde à linha real da planilha.

### Colunas de stations

Aceita os nomes originais (`SITE ID`, `TECNOLOGIA`, `MUNICIPIO`, etc.) e os
nomes da API (`site_id`, `technology`, `city`, etc.):

```text
SITE ID, TECNOLOGIA, END ID, TIPO_DE_CONTRATO_AREA, DETENTOR_AREA,
TIPO_DE_INFRA, TIPO_DE_EV, ENDERECO, BAIRRO, MUNICIPIO, ESTADO, CEP,
REGIONAL, LATITUDE, LONGITUDE, STATUS, TIPO_DA_TORRE, AEV_NOMINAL,
STATION_ID, FORNECEDOR_DE_EV, AREA_DO_SOLO, ALTURA_DA_ESTRUTURA,
OBSERVACAO_THQ
```

`SITE ID`, `LATITUDE` e `LONGITUDE` são as únicas colunas obrigatórias. Números
aceitam vírgula decimal
(`-18,345528`) e separador de milhar com ponto (`18.000,5`).

### Colunas de radio links

```text
LINK_ID, NAME, STATION_A_SITE_ID, STATION_B_SITE_ID, TECHNOLOGY,
FREQUENCY_MHZ, CAPACITY_MBPS, DISTANCE_KM, STATUS, OBSERVATION
```

Obrigatórias: `LINK_ID`, `STATION_A_SITE_ID`, `STATION_B_SITE_ID`.

> Para preservar zeros à esquerda no CEP, formate a coluna como texto no Excel.

### Jobs para arquivos grandes

Arquivos com centenas de milhares de linhas devem usar os endpoints assíncronos:

| Método | Rota | Descrição |
| --- | --- | --- |
| `POST` | `/api/v1/stations/import/jobs` | Enfileira importação de stations |
| `POST` | `/api/v1/radio-links/import/jobs` | Enfileira importação de radio links |
| `GET` | `/api/v1/import-jobs/{jobID}` | Consulta status e progresso |
| `GET` | `/api/v1/import-jobs` | Lista jobs |

Exemplo:

```bash
curl -X POST "http://localhost:8080/api/v1/stations/import/jobs?mode=upsert" \
  -F "file=@stations-700k.xlsx"
```

Resposta `202 Accepted`:

```json
{
  "id": "6f0e6e6e-...",
  "entity": "stations",
  "status": "pending",
  "mode": "upsert",
  "dry_run": false,
  "file_name": "stations-700k.xlsx",
  "total_rows": 0,
  "processed_rows": 0,
  "valid_rows": 0,
  "imported_rows": 0,
  "updated_rows": 0,
  "failed_rows": 0,
  "errors": []
}
```

Acompanhe:

```bash
curl "http://localhost:8080/api/v1/import-jobs/6f0e6e6e-..."
```

Status: `pending`, `running`, `completed`, `failed` ou `canceled`.

Características:

- Leitura do Excel em streaming, sem carregar todas as linhas na memória
- Processamento em lotes configuráveis (`IMPORT_BATCH_SIZE`)
- Upsert em lote via GORM
- Progresso persistido a cada lote
- Erros limitados por job (`IMPORT_ERROR_LIMIT`), mantendo a contagem total
- Job interrompido por restart é marcado como `failed`
- Máximo de upload: `IMPORT_MAX_BYTES` (padrão 250 MiB)
- Diretório dos arquivos: `IMPORT_DIR` (padrão `./data/imports`)
- Workers: `IMPORT_WORKERS` (padrão `1`)
- Jobs `pending` são retomados na inicialização

## Validação

Todos os endpoints validam a entrada de forma consistente:

| Regra | Resposta |
| --- | --- |
| `POST`/`PUT` sem `Content-Type: application/json` | `415 unsupported_media_type` |
| Body maior que 1 MiB | `413 payload_too_large` |
| JSON inválido, campo desconhecido ou múltiplos objetos | `400 invalid_request` |
| Campo obrigatório ausente ou inválido | `400 validation_error` |
| `page < 1` ou `page_size` fora de `1..100` | `400 invalid_request` |
| `search` acima de 255 caracteres | `400 invalid_request` |
| `site_id`/`link_id`/`station_id` acima de 50 caracteres | `400 validation_error` ou `400 invalid_request` |
| Link referenciando estação inexistente | `422 station_not_found` |
| Remoção de estação usada por link | `409 station_in_use` |
| Recurso inexistente | `404 not_found` |
| Chave duplicada | `409 already_exists` |

Regras de domínio:

- Station: `site_id`, `latitude` e `longitude` obrigatórios; UF válida e
  coordenadas dentro dos limites; área/altura não negativas
- Radio link: pontas A e B obrigatórias e diferentes, frequência e capacidade
  positivas, distância não negativa
- `GET`/`DELETE` validam o identificador de path
- `POST`/`PUT` rejeitam campos desconhecidos do JSON

## Migrações e seed

As migrações são embutidas no binário e aplicadas automaticamente ao iniciar a
API. Para executar manualmente:

```bash
make migrate
# ou
go run ./cmd/migrate
```

O seed de desenvolvimento é idempotente e aplicado nesta ordem:

- `scripts/seed.sql`: 15 stations de exemplo
- `scripts/seed_radio_links.sql`: 7 radio links entre essas stations

```bash
make seed
# ou
go run ./cmd/seed
```

Para usar apenas um arquivo customizado:

```bash
SEED_FILE=./meu-seed.sql go run ./cmd/seed
```

## Swagger

Depois de subir a API:

- UI: `http://localhost:8080/swagger/index.html`
- OpenAPI JSON: `http://localhost:8080/swagger/doc.json`

Para regenerar a documentação após alterar anotações:

```bash
make swagger
```

## Testes

```bash
# Todos os testes (unitários + repositórios com SQLite em memória)
go test ./...

# Com detector de corrida (requer CGO/gcc, ex.: Linux/macOS ou MinGW no Windows)
go test -race ./...

# Cobertura
make cover
```

Os testes de repositório usam SQLite em memória (`modernc.org/sqlite`) e não
precisam de MySQL. O MySQL local continua sendo usado apenas para rodar a
aplicação em desenvolvimento.

## Banco de dados e adapters

O domínio e os casos de uso não conhecem MySQL nem GORM.

- **Status:** usa a porta `DatabasePinger`; o adapter `sqlpinger` funciona com
  qualquer driver de `database/sql`.
- **Stations:** usa a porta `station.Repository`; o adapter `stationmysql` usa
  GORM.
- **Radio links:** usa a porta `radiolink.Repository`; o adapter
  `radiolinkmysql` usa GORM.
- **Modelos GORM:** ficam em `internal/adapter/gormmodel`.
- **Testes:** usam GORM com SQLite em memória.

Para trocar de banco:

1. Adicione o dialector GORM correspondente em `database.Open`.
2. Adicione migrações compatíveis com o novo dialeto.
3. Ajuste `DB_DRIVER`/`DB_DSN` e conecte o dialector no `main`.

Nenhuma camada de domínio, caso de uso ou HTTP precisa mudar.

## Configuração

Todas as variáveis estão documentadas em `.env.example`. As principais:

| Variável | Padrão | Descrição |
| --- | --- | --- |
| `APP_NAME` | `api-stations-and-radio-link` | Nome exibido no status |
| `APP_VERSION` | `1.0.0` | Versão exibida no status |
| `APP_ENV` | `development` | Ambiente (`development` usa log texto) |
| `HTTP_ADDR` | `:8080` | Endereço do servidor HTTP |
| `SWAGGER_HOST` | `localhost:8080` | Host exibido no Swagger |
| `DB_DRIVER` | `mysql` | Dialeto GORM (`mysql`) |
| `DB_DSN` | DSN MySQL montado | Sobrescreve o DSN completo |
| `DB_HOST` / `DB_PORT` | `127.0.0.1` / `3306` | Endereço do MySQL |
| `DB_USER` / `DB_PASSWORD` | `root` / `admin` | Credenciais do MySQL |
| `DB_NAME` | `api_stations_radio_link` | Nome do banco |
| `DB_PING_TIMEOUT` | `2s` | Timeout do ping de status |
| `IMPORT_DIR` | `./data/imports` | Diretório dos arquivos de importação |
| `IMPORT_MAX_BYTES` | `262144000` | Limite de upload dos jobs |
| `IMPORT_BATCH_SIZE` | `1000` | Linhas por lote (máx. 1000) |
| `IMPORT_ERROR_LIMIT` | `1000` | Erros armazenados por job |
| `IMPORT_WORKERS` | `1` | Workers de importação |

## CI/CD e deploy com PM2

O projeto tem dois workflows:

- `.github/workflows/ci.yml`: roda em `push`/`pull_request` para `main`, executa
  `gofmt`, `go vet`, valida o Swagger, roda testes unitários e de integração com
  SQLite e gera o binário Linux.
- `.github/workflows/deploy.yml`: roda automaticamente após o CI concluir com
  sucesso em `main` (ou manualmente via **Run workflow**). Ele compila o binário
  Linux, envia para o servidor via SSH e executa `deploy.sh`.

### Como o `deploy.sh` funciona

1. Copia o binário para `releases/<release-id>/`.
2. Atualiza o ponteiro `current_release` e o binário estável usado pelo PM2.
3. Executa `pm2 startOrReload ecosystem.config.cjs --update-env` e `pm2 save`.
4. Valida `GET /api/v1/healthz` (liveness). Por compatibilidade, também aceita
   um `503` de `/api/v1/status`, que indica processo no ar mas banco indisponível.
5. Em caso de falha, faz rollback para a release anterior.
6. Mantém as 5 releases mais recentes (`KEEP_RELEASES`).

### Configuração do servidor

Produção atual: `root@russian:~/app/api-stations-and-radio-link`.

Pré-requisitos no servidor:

- Go não é necessário no servidor; o binário é compilado no GitHub Actions.
- Node.js + PM2 instalados para o usuário `root`:

```bash
npm install -g pm2
pm2 startup
pm2 save
```

- Diretório de deploy:

```bash
mkdir -p /root/app/api-stations-and-radio-link/{incoming,releases,logs}
```

- Arquivo `/root/app/api-stations-and-radio-link/.env` com as variáveis de
  produção (use `.env.example` como base). O PM2 usa `cwd` nesse diretório e a
  aplicação carrega o `.env` automaticamente.
- Chave SSH do usuário `root` cadastrada no GitHub.

> `russian` precisa ser resolvido pelo runner do GitHub Actions. Se ele for
> apenas um alias do seu `~/.ssh/config` local, cadastre em `DEPLOY_HOST` o IP
> ou domínio real do servidor.

O primeiro deploy cria a release, inicia o processo no PM2 e salva a lista de
processos. Os logs ficam em `/root/app/api-stations-and-radio-link/logs/` e o
ponteiro da release ativa em `current_release`.

Comandos úteis no servidor:

```bash
pm2 status
pm2 logs api-stations-and-radio-link
pm2 restart api-stations-and-radio-link
```

Se `GET /api/v1/status` retornar `503`, o processo está no ar mas o banco não
responde. Verifique o `.env` em `/root/app/api-stations-and-radio-link/.env`,
se o MySQL está rodando e os logs do PM2. Após corrigir, reinicie:

```bash
pm2 restart api-stations-and-radio-link --update-env
```

### Secrets do GitHub

Configure em **Settings → Secrets and variables → Actions** (ou no environment
`production`):

| Secret | Obrigatório | Padrão | Descrição |
| --- | --- | --- | --- |
| `DEPLOY_HOST` | não | `russian` | IP, domínio ou hostname resolvível pelo runner |
| `DEPLOY_USER` | não | `root` | Usuário SSH de deploy |
| `DEPLOY_SSH_KEY` | sim | — | Chave privada SSH |
| `DEPLOY_KNOWN_HOSTS` | recomendado | `ssh-keyscan` | Saída de `ssh-keyscan -H host` |
| `DEPLOY_PORT` | não | `22` | Porta SSH |
| `DEPLOY_PATH` | não | `/root/app/api-stations-and-radio-link` | Diretório de deploy |
| `PM2_APP_NAME` | não | `api-stations-and-radio-link` | Nome do processo no PM2 |
| `HEALTHCHECK_URL` | não | `http://127.0.0.1:8080/api/v1/healthz` | URL de healthcheck |

Para gerar `DEPLOY_KNOWN_HOSTS`:

```bash
ssh-keyscan -H russian
# ou
ssh-keyscan -H <ip-ou-dominio>
```

### Deploy manual

O script é executado no servidor (o workflow já faz esse passo). Para rodar
manualmente, envie o binário e o `deploy.sh` para `DEPLOY_PATH/incoming/` e
execute:

```bash
DEPLOY_DIR=/root/app/api-stations-and-radio-link \
PM2_APP_NAME=api-stations-and-radio-link \
BIN_SOURCE=/root/app/api-stations-and-radio-link/incoming/api \
bash /root/app/api-stations-and-radio-link/incoming/deploy.sh
```

## Próximos passos

- `PATCH` para atualização parcial de stations e radio links
- Filtros por estado, status e tecnologia
- Cálculo automático de distância entre as pontas
- Autenticação/autorização
- Paginação por cursor para grandes volumes
