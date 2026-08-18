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
echo "JWT_SECRET=troque-isso" > .env
go mod tidy
go run main.go
```

Servidor em `http://localhost:8080`. Coleção: `requests.http`.

## Arquitetura

`handler` → `service` → `repository`
