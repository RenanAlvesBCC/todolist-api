# Oficina API

Antes de escrever código:

1. Leia as rules em `.cursor/rules/` (inclui **commits-e-cobertura**).
2. Siga o skill `.cursor/skills/implementar-feature/SKILL.md` em todo incremento.
3. Consulte `docs/VISAO.md`, `docs/GLOSSARIO.md`, `docs/CONTRATO-API.md`, `docs/ROADMAP.md` e `docs/QUALIDADE.md`.
4. Antes de cada commit: `./scripts/verify.sh` (build + testes + cobertura ≥ 90% no escopo).

Clientes: `oficina-web` / `oficina-gerente-web` (gerente) e `todolist-ios` / `oficina-mecanico-ios` + Android (mecânico).
