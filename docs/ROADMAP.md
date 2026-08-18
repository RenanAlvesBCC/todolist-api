# Roadmap

Visão compartilhada dos quatro repositórios.

## Feito neste ciclo

- [x] Fase -1 — AGENTS.md, `.cursor/rules`, skill `implementar-feature`
- [x] Fase 0 — VISÃO, GLOSSÁRIO, ROADMAP, CONTRATO-API
- [x] Fase 2 — board do mecânico na API (`status`, `mine`, role, restrições, auditoria)
- [x] Fase 3 — pivot iOS para esteira do mecânico
- [x] Fase 4 — scaffold Android (Compose)
- [x] Fase 5 — web com role real, convite público, copy de gerente
- [x] Identidade local — bundle/applicationId, display name, `package.json`, pastas `oficina-api` e `oficina-gerente-web`
- [ ] Fase 1 (GitHub) — `gh auth login` e rename dos remotes: `oficina-api`, `oficina-gerente-web`, `oficina-mecanico-ios`; criar `oficina-mecanico-android`. Pasta local `todolist-ios` → `oficina-mecanico-ios` (ver `docs/RENOMEAR-GITHUB.md`)

## Backlog

- Renomear HTTP `/api/lists` → `/api/vehicles` nos quatro repos no mesmo PR
- Renomear structs Go `TaskList` / `TaskItem` (tabelas podem usar `TableName()`)
- Refresh de JWT (hoje 24h)
- Role API `mechanic` no lugar de `editor`
- Publicação App Store / Play Store

## Como incrementar

Todo código novo segue `.cursor/skills/implementar-feature` e as rules do repo. Não misturar visão de gerente com visão de mecânico.
