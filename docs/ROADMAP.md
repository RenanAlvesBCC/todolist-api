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
- [ ] Fase 1 (GitHub) — `gh auth login` e rename dos remotes: `oficina-api`, `oficina-gerente-web`, `oficina-mecanico-ios`. Pasta local `todolist-ios` → `oficina-mecanico-ios` (ver `docs/RENOMEAR-GITHUB.md`)
- [x] Android no GitHub como `oficina-mecanico-android`
- [x] CI `verify.sh` nos quatro repos
- [x] Polish qualidade (cobertura ≥ 90% no escopo) + paridade UX desteira
- [x] API — base de clientes, veículos do cliente, histórico de OS, vínculo do atendimento (`customer_id` / `customer_vehicle_id`), tokens de integração e importação OnMotor (mão única)

## Backlog

- Web do gerente: telas de clientes, veículos do cliente, histórico, reconhecimento por placa ao criar atendimento, tokens de integração, upload de planilha OnMotor e status das sincronizações
- Robô de sincronização OnMotor (só leitura no OnMotor) usando `POST /api/integrations/onmotor/import`

- Renomear HTTP `/api/lists` → `/api/vehicles` nos quatro repos no mesmo PR
- Renomear structs Go `TaskList` / `TaskItem` (tabelas podem usar `TableName()`)
- Refresh de JWT (hoje 24h)
- Role API `mechanic` no lugar de `editor`
- Publicação App Store / Play Store

## Como incrementar

Todo código novo segue `.cursor/skills/implementar-feature` e as rules do repo. Não misturar visão de gerente com visão de mecânico.
