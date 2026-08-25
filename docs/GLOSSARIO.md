# Glossário

Termos de negócio vs nomes técnicos atuais (legado to-do). A UI e a documentação usam a coluna da esquerda. O HTTP ainda usa a da direita até o rename `/api/vehicles`.

| Negócio | Código / HTTP |
| --- | --- |
| Oficina | `Workspace` |
| Dono | `owner` |
| Gerente | `manager` |
| Mecânico | `editor` |
| Veículo / carro na esteira | `TaskList`, `/api/lists` |
| Placa | `plate` |
| Cliente | `customer` |
| Serviço (item de trabalho) | `TaskItem`, `/api/lists/:id/items` |
| Mecânico atribuído | `ListAssignment`, `/assignments` |
| Linha de orçamento | `QuoteItem`, `/quotes` |
| Pendência | `PendingFlag`, `/flags` |
| Esteira do mecânico | `GET /api/lists?mine=true` (editor sempre filtrado) |
| Pátio (visão gerente) | `GET /api/lists` sem `mine` |

## Status do veículo

- `aguardando_box`
- `em_diagnostico`
- `aguardando_orcamento`
- `em_andamento`
- `aguardando_peca`
- `test_drive`
- `lava_jato`
- `aguardando_checkup`
- `aguardando_retirada`
- `entregue`

Fluxo linear é sugestão de UI; mecânico e gerente podem setar qualquer status válido.

Legado (migrado no boot): `aprovado` → `aguardando_retirada`; `concluido` → `entregue`.

## Tipos de pendência

`aguardando_ml`, `procurando_peca`, `aguardando_cliente`, `aguardando_orcamento`, `aguardando_entrega`, `aguardando_retirada`, `aguardando_terceiro`, `outro`.
