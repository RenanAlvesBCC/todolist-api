# Visão do produto

Sistema de oficina mecânica com **uma API** e **dois tipos de cliente**.

## Papéis

| API (`role`) | Na UI | Onde trabalha |
| --- | --- | --- |
| `owner` | Dono | Web do gerente |
| `manager` | Gerente | Web do gerente |
| `editor` | Mecânico | App iOS e Android |

## O que cada um faz

**Dono / gerente (web)**

- Vê todos os veículos da oficina (pátio).
- Cria, edita título, exclui e reordena veículos.
- Atribui e remove mecânicos de um carro.
- Muda qualquer status.
- Gerencia membros e convites (dono).
- Consulta auditoria.

**Mecânico (app)**

- Vê **somente** os carros da sua esteira (atribuídos a ele).
- Atualiza status nas transições permitidas: `em_andamento`, `aguardando_orcamento`, `aguardando_peca` (não aprova nem conclui).
- Marca serviços, registra pendências e linhas de orçamento.
- Não cria veículo, não atribui colegas, não gerencia membros.

## Repositórios

- `oficina-api` (hoje `todolist-api`) — backend
- `oficina-gerente-web` (hoje `oficina-web`) — painel do gerente
- `oficina-mecanico-ios` (hoje `todolist-ios`) — app do mecânico
- `oficina-mecanico-android` — app do mecânico (Kotlin)

## Documentos

- [GLOSSARIO.md](./GLOSSARIO.md)
- [ROADMAP.md](./ROADMAP.md)
- [CONTRATO-API.md](./CONTRATO-API.md)
