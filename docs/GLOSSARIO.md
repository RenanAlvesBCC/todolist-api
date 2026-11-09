# Glossário

Termos de negócio vs nomes técnicos atuais (legado to-do). A UI e a documentação usam a coluna da esquerda. O HTTP ainda usa a da direita até o rename `/api/vehicles`.

| Negócio | Código / HTTP |
| --- | --- |
| Oficina | `Workspace` |
| Dono | `owner` |
| Gerente | `manager` |
| Mecânico | `editor` |
| Veículo / carro na esteira | `TaskList`, `/api/lists` |
| Atendimento (uma passagem do carro pela oficina; hoje cada veículo em `/api/lists` é um atendimento, vinculável ao cadastro por `customer_id` / `customer_vehicle_id`) | `TaskList`, `/api/lists` |
| Placa | `plate` |
| Nome do cliente no atendimento (texto livre) | `customer` |
| Cliente (cadastro permanente da oficina, com documentos e contatos — LGPD; só dono/gerente) | `Customer`, `/api/customers` |
| Veículo do cliente (cadastro permanente do carro; placa única por oficina) | `CustomerVehicle`, `/api/customers/:id/vehicles`, `/api/customer-vehicles` |
| Histórico do veículo (OS do OnMotor + nossos atendimentos, mais recentes primeiro) | `ServiceRecord` + `TaskList`, `GET /api/customer-vehicles/:id/history` |
| Origem do cadastro (`local` = criado aqui; `onmotor` = importado, `source_id` = id no OnMotor) | `source`, `source_id` |
| Quilometragem (km rodados do veículo) | `mileage` (inteiro ou `null`) |
| Dados do cliente (CPF, CEP, WhatsApp — LGPD; só dono/gerente veem) | `customer_cpf`, `customer_cep`, `customer_whatsapp` (só dígitos) |
| Serviço (item de trabalho) | `TaskItem`, `/api/lists/:id/items` |
| Mecânico atribuído | `ListAssignment`, `/assignments` |
| Linha de orçamento | `QuoteItem`, `/quotes` |
| Pendência | `PendingFlag`, `/flags` |
| Informação importante (nota do veículo, com data/hora e autor) | `VehicleNote`, `/notes` |
| Exclusão lógica (some da tela, fica no histórico) | `deleted_at` + `deleted_by` |
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

Data de entrega: `delivered_at` no veículo, preenchido ao entrar em `entregue` e limpo ao sair.

## Prioridade do veículo

`priority`: `normal` (default) | `alta` | `urgente`. Definida pelo dono/gerente (`PUT /api/lists/:id/priority` ou na ficha); mecânico só vê. Não muda a ordem do pátio (continua por `position`).

## Quilometragem

`mileage`: km do veículo, inteiro de `0` a `2000000`, ou `null` quando não informada (inclusive veículos antigos). Dono/gerente informam na criação ou na ficha (`PUT /api/lists/:id`); todos os papéis veem. Na ficha, omitir (ou mandar `null`) mantém o valor; não dá para voltar a `null`, só sobrescrever.

## Vídeo de check-up

Toda entrada de veículo exige gravar um vídeo de check-up. A UI mostra o lembrete fixo "Vídeo de check-up feito?" enquanto `checkup_video_done_at` for `null`. Marcar: `PUT /api/lists/:id/checkup-video` `{ "done": true }` (dono, gerente ou mecânico atribuído); grava data/hora e autor (`checkup_video_done_by`, `checkup_video_done_by_name`). `{ "done": false }` desfaz.

## Tipos de pendência

`aguardando_ml`, `procurando_peca`, `aguardando_cliente`, `aguardando_orcamento`, `aguardando_entrega`, `aguardando_retirada`, `aguardando_terceiro`, `outro`.
