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
- PM2 em produção
- GitHub Actions para CI/CD

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

## CI/CD e deploy com PM2

O projeto tem dois workflows:

- `.github/workflows/ci.yml`: roda em `push`/`pull_request` para `main`, executa
  `gofmt`, `go vet`, valida o Swagger, roda testes unitários e de integração com
  MySQL e gera o binário Linux.
- `.github/workflows/deploy.yml`: roda automaticamente após o CI concluir com
  sucesso em `main` (ou manualmente via **Run workflow**). Ele compila o binário
  Linux, envia para o servidor via SSH e executa `deploy.sh`.

### Como o `deploy.sh` funciona

1. Copia o binário para `releases/<release-id>/`.
2. Atualiza o ponteiro `current_release` e o binário estável usado pelo PM2.
3. Executa `pm2 startOrReload ecosystem.config.cjs --update-env` e `pm2 save`.
4. Valida `GET /api/v1/status` (aceita qualquer HTTP < 500; um `503` pode
   significar apenas banco indisponível).
5. Em caso de falha, faz rollback para a release anterior.
6. Mantém as 5 releases mais recentes (`KEEP_RELEASES`).

### Configuração do servidor

Pré-requisitos no servidor:

- Go não é necessário no servidor; o binário é compilado no GitHub Actions.
- Node.js + PM2 instalados para o usuário de deploy:

```bash
npm install -g pm2
pm2 startup
pm2 save
```

- Diretório de deploy com permissão do usuário de deploy:

```bash
sudo mkdir -p /opt/api-stations-and-radio-link/{incoming,releases,logs}
sudo chown -R deploy:deploy /opt/api-stations-and-radio-link
```

- Arquivo `/opt/api-stations-and-radio-link/.env` com as variáveis de produção
  (use `.env.example` como base). O PM2 usa `cwd` nesse diretório e a aplicação
  carrega o `.env` automaticamente.
- Chave SSH do usuário de deploy cadastrada no GitHub.

O primeiro deploy cria a release, inicia o processo no PM2 e salva a lista de
processos. Os logs ficam em `/opt/api-stations-and-radio-link/logs/` e o ponteiro
da release ativa em `current_release`.

Comandos úteis no servidor:

```bash
pm2 status
pm2 logs api-stations-and-radio-link
pm2 restart api-stations-and-radio-link
```

### Secrets do GitHub

Configure em **Settings → Secrets and variables → Actions** (ou no environment
`production`):

| Secret | Obrigatório | Padrão | Descrição |
| --- | --- | --- | --- |
| `DEPLOY_HOST` | sim | — | IP ou domínio do servidor |
| `DEPLOY_USER` | sim | — | Usuário SSH de deploy |
| `DEPLOY_SSH_KEY` | sim | — | Chave privada SSH |
| `DEPLOY_KNOWN_HOSTS` | recomendado | `ssh-keyscan` | Saída de `ssh-keyscan -H host` |
| `DEPLOY_PORT` | não | `22` | Porta SSH |
| `DEPLOY_PATH` | não | `/opt/api-stations-and-radio-link` | Diretório de deploy |
| `PM2_APP_NAME` | não | `api-stations-and-radio-link` | Nome do processo no PM2 |
| `HEALTHCHECK_URL` | não | `http://127.0.0.1:8080/api/v1/status` | URL de healthcheck |

Para gerar `DEPLOY_KNOWN_HOSTS`:

```bash
ssh-keyscan -H seu-servidor.com
```

### Deploy manual

O script é executado no servidor (o workflow já faz esse passo). Para rodar
manualmente, envie o binário e o `deploy.sh` para `DEPLOY_PATH/incoming/` e
execute:

```bash
DEPLOY_DIR=/opt/api-stations-and-radio-link \
PM2_APP_NAME=api-stations-and-radio-link \
BIN_SOURCE=/opt/api-stations-and-radio-link/incoming/api \
bash /opt/api-stations-and-radio-link/incoming/deploy.sh
```

## Próximos passos

- Endpoints de estações e enlace de rádio
- Migrations versionadas
- Autenticação/autorização
- Health check separado de readiness/liveness
