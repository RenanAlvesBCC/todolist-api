# oficina-api

API REST da oficina (Go + Gin + GORM + SQLite). Backend único do painel do gerente e dos apps do mecânico.

## Documentação

- [Visão](docs/VISAO.md)
- [Glossário](docs/GLOSSARIO.md)
- [Roadmap](docs/ROADMAP.md)
- [Contrato](docs/CONTRATO-API.md)

Antes de implementar: `AGENTS.md` e `.cursor/skills/implementar-feature`.

## Como rodar

```bash
cp .env.example .env
# Edite .env: JWT_SECRET e, se for usar Neon/Postgres, DATABASE_URL
go mod tidy
go run main.go
```

- Sem `DATABASE_URL`: SQLite em `app.db`.
- Com `DATABASE_URL` (ex.: Neon): Postgres; no boot o `AutoMigrate` cria as tabelas.

Servidor em `http://localhost:8080`. Coleção: `requests.http`.

### Produção (Render + Neon)

No Web Service da API, defina:

| Variável | Valor |
|----------|--------|
| `JWT_SECRET` | segredo forte |
| `DATABASE_URL` | connection string **pooled** do Neon (`…-pooler…?sslmode=require`) |

Depois faça redeploy. Clientes (web/iOS/Android) continuam apontando para a URL da API no Render.

## Arquitetura

`handler` → `service` → `repository`
