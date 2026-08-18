# Qualidade — oficina-api

## Escopo de cobertura (meta ≥ 90%)

| Incluído | Excluído |
|----------|----------|
| `internal/services/` | `main.go` |
| `internal/handlers/` | `internal/routes/` |
| `internal/repository/` | `internal/database/`, `internal/middleware/`, `internal/models/`, `internal/utils/` |

Medição: `go test` com `-coverprofile` nos três pacotes acima; total agregado via `go tool cover -func`.

## Verificação local

```bash
./scripts/verify.sh
```

Equivalente manual:

```bash
go build ./...
go test ./internal/services/... ./internal/handlers/... ./internal/repository/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## Commits pequenos

- Um commit = handler, service, repository ou fix de teste — não misture refactors amplos.
- `./scripts/verify.sh` deve passar antes de commitar.
- Todo handler/service/repository novo exige `*_test.go` no mesmo commit.

## Baseline

Se a cobertura agregada estiver abaixo de 90%, cada incremento no escopo deve incluir testes até atingir o threshold.
