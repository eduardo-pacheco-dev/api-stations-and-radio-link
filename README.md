# API Stations and Radio Link

API Go para gerenciamento de estações e enlace de rádio. Nesta primeira versão
existe apenas o endpoint de status da API, com verificação de conectividade com
o banco de dados.

## Stack

- Go 1.24+
- `net/http` (sem framework web)
- MySQL 8.4
- Swagger/OpenAPI via `swaggo/swag` + `http-swagger`
- `database/sql` com driver MySQL
- `log/slog` para logs estruturados

## Estrutura

```text
.
├── cmd/api/main.go                         # composition root, servidor e shutdown
├── docs/                                   # documentação Swagger gerada
├── internal/
│   ├── adapter/
│   │   ├── database/                       # abertura e pool do database/sql
│   │   ├── httpapi/                        # rotas, handler, middleware
│   │   └── sqlpinger/                      # adapter concreto de ping no banco
│   ├── config/                             # configuração via variáveis de ambiente
│   └── status/                             # caso de uso + portas + read model
├── docker-compose.yml                      # MySQL local (root/admin)
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

1. Suba o MySQL:

```bash
docker compose up -d mysql
```

O container cria o banco `api_stations_radio_link` com usuário `root` e senha
`admin`.

Se já existir um MySQL local sem o banco, crie-o uma vez:

```bash
mysql -uroot -padmin < scripts/init.sql
```

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
# Unitários
go test ./...

# Com detector de corrida (requer CGO/gcc, ex.: Linux/macOS ou MinGW no Windows)
go test -race ./...

# Integração (requer MySQL)
go test -tags=integration ./...

# Cobertura
make cover
```

Os testes de integração usam `TEST_DB_DSN` quando definido. Caso contrário,
tentam `root:admin@tcp(127.0.0.1:3306)/api_stations_radio_link` e são ignorados
(`skip`) se o MySQL não estiver disponível.

## Banco de dados desacoplado

O core da aplicação não conhece MySQL. Ele depende da porta:

```go
type DatabasePinger interface {
    Ping(ctx context.Context) error
    Driver() string
}
```

O adapter `internal/adapter/sqlpinger` implementa essa porta sobre qualquer
driver de `database/sql`. Para trocar de banco:

1. Registre o novo driver no composition root (`cmd/api/main.go`):

```go
import _ "github.com/jackc/pgx/v5/stdlib"
```

2. Ajuste as variáveis de ambiente:

```env
DB_DRIVER=pgx
DB_DSN=postgres://user:password@localhost:5432/stations?sslmode=disable
```

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
| `DB_DRIVER` | `mysql` | Driver registrado em `database/sql` |
| `DB_DSN` | DSN MySQL montado | Sobrescreve o DSN completo |
| `DB_HOST` / `DB_PORT` | `127.0.0.1` / `3306` | Endereço do MySQL |
| `DB_USER` / `DB_PASSWORD` | `root` / `admin` | Credenciais do MySQL |
| `DB_NAME` | `api_stations_radio_link` | Nome do banco |
| `DB_PING_TIMEOUT` | `2s` | Timeout do ping de status |

## Próximos passos

- Endpoints de estações e enlace de rádio
- Migrations versionadas
- Autenticação/autorização
- Health check separado de readiness/liveness
- CI com `go test -race` e `go test -tags=integration`
