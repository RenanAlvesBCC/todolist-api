---
name: implementar-feature
description: Guides implementing a product increment for the oficina (mechanic board, manager web, or API). Use when adding a feature, endpoint, screen, or when the user asks to implementar, incrementar, pivotar, or follow the roadmap.
---

# Implementar feature — oficina-api

Este repo é a **API** compartilhada. Clientes: web do gerente e apps do mecânico.

## Checklist

```
- [ ] Li docs/VISAO.md, docs/GLOSSARIO.md, docs/CONTRATO-API.md e as rules
- [ ] Classifiquei: API | web gerente | app mecânico | contrato cruzado
- [ ] Confirmei o papel (owner / manager / editor). editor = mecânico
- [ ] Contrato HTTP documentado em docs/CONTRATO-API.md
- [ ] Código na camada certa (handler → service → repository)
- [ ] Testes *_test.go no mesmo incremento
- [ ] Erros de negócio em português via RespondError
- [ ] `./scripts/verify.sh` passou (build + testes + cobertura ≥ 90% no escopo)
```

## Antes do commit

1. Commit **pequeno** — handler, service ou repository por vez.
2. Rode `./scripts/verify.sh`; se falhar, adicione testes ou corrija antes de commitar.
3. Escopo: `internal/services`, `internal/handlers`, `internal/repository` — ver `docs/QUALIDADE.md`.

## Regras

1. Não ampliar permissão do mecânico sem mudança explícita no contrato.
2. Não inventar endpoint, campo JSON ou role sem documentar.
3. `editor` só vê veículos atribuídos; create/delete/reorder/assign só owner/manager.
4. Handler não fala com GORM. Service não monta status HTTP.

## Depois

Atualize `docs/CONTRATO-API.md` e `requests.http`. Avise iOS, Android e web se o JSON mudou.
