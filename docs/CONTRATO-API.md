# Contrato da API

Base: `http://localhost:8080` (dev) ou a URL de produção no Render.

Autenticação: `Authorization: Bearer <jwt>` nas rotas `/api`. Token expira em 24h. Logout coloca o token na blacklist.

No JSON, `lists` = veículos e `items` = serviços. Ver [GLOSSARIO.md](./GLOSSARIO.md).

## Públicos

### POST /register
Body: `{ "username": "email", "password": "string", "first_name": "string", "last_name": "string" }`
- `username` é o e-mail de login.
- 201 `{ "message": "usuário criado com sucesso" }`
- 400 dados inválidos / nome ou e-mail em branco
- 409 usuário já existe

### POST /login
Body: `{ "username": "email", "password": "string" }`
- 200 `{ "token": "<jwt>" }`

### GET /invites/:code/preview
Prévia do convite (oficina + role) sem login.

## Auth

### POST /api/logout
Invalida o token atual. 204.

## Workspace

### POST /api/workspace
Cria a oficina. Body: `{ "name", "description?" }`. Só uma por conta.

### GET /api/workspace
Oficina do membro autenticado, **com o papel atual**:

```json
{
  "id": 1,
  "name": "Oficina Centro",
  "description": "",
  "owner_id": 1,
  "role": "editor"
}
```

`role` é `owner` | `manager` | `editor`.

### PUT /api/workspace
Atualiza nome/descrição (dono).

### POST /api/workspace/invites
Body: `{ "role": "manager" | "editor" }`. Só dono. Resposta `{ "code": "..." }`.

### GET /api/workspace/invites
Lista convites (dono).

### POST /api/invites/:code/accept
Entra na oficina.

### GET /api/workspace/members
Lista membros (todos os papéis). Cada item inclui o `user` associado.

```json
[
  {
    "id": 1,
    "workspace_id": 1,
    "user_id": 7,
    "role": "editor",
    "joined_at": "2026-08-10T14:00:00Z",
    "last_seen_at": null,
    "user": {
      "id": 7,
      "username": "mecanico@teste.com",
      "first_name": "Ana",
      "last_name": "Silva",
      "created_at": "...",
      "updated_at": "..."
    }
  }
]
```

- `password` nunca é retornado.
- `first_name` / `last_name` podem ser `""` em contas antigas (antes do cadastro com nome).

### DELETE /api/workspace/members/:userId
Remove membro (dono).

## Veículos (`/api/lists`)

### GET /api/lists
Query: `search`, `page`, `limit`, `status`, `mine=true`. `search` casa com `title`, `plate` ou `customer`.

- `owner` / `manager`: pátio inteiro (filtros opcionais).
- `editor`: **somente veículos em que está atribuído** (mesmo sem `mine`).
- `mine=true`: restringe a atribuídos também para gerente.

Resposta:

```json
{
  "lists": [ /* TaskList / veículo */ ],
  "page": 1,
  "limit": 20,
  "total": 4,
  "total_pages": 1
}
```

Status: `aguardando_box` | `em_diagnostico` | `aguardando_orcamento` | `em_andamento` | `aguardando_peca` | `test_drive` | `lava_jato` | `aguardando_checkup` | `aguardando_retirada` | `entregue`.

Default ao criar veículo: `aguardando_box`.

Legado: `aprovado` → `aguardando_retirada`; `concluido` → `entregue` (migração no boot).

### POST /api/lists
Cria veículo. Só `owner`/`manager`. Body:

```json
{
  "title": "Gol 2012",
  "plate": "abc1d23",
  "customer": "Maria Souza",
  "customer_cpf": "123.456.789-09",
  "customer_cep": "01310-100",
  "customer_whatsapp": "+55 (11) 98765-4321",
  "priority": "alta",
  "mileage": 87500,
  "customer_id": 12,
  "customer_vehicle_id": 30
}
```

Só `title` é obrigatório (exceto com `customer_vehicle_id`, ver [Vínculo com o cadastro de clientes](#vínculo-com-o-cadastro-de-clientes)); os demais têm default `""` (`priority`: default `normal`; valor fora do enum → 400 `prioridade inválida`; `mileage`: default `null`, regras em [Quilometragem](#quilometragem)). Placa é normalizada (trim + maiúsculas), sem validação de formato. CPF/CEP/WhatsApp seguem as regras de [Dados do cliente](#dados-do-cliente-lgpd). 201 com o veículo (visão de dono/gerente). 400 em dado inválido.

### GET /api/lists/:id
Detalhe. Mecânico só se atribuído ou membro com acesso ao veículo da esteira.

### PUT /api/lists/:id
Ficha do veículo. Só `owner`/`manager`; mecânico só lê (403).

- `title` (obrigatório), `plate` e `customer` são **substituídos** sempre (como antes).
- `customer_cpf`, `customer_cep`, `customer_whatsapp` são opcionais: **chave ausente (ou `null`) mantém o valor atual**; chave presente grava, e `""` limpa.
- `priority` também é opcional: ausente (ou `null`) mantém; presente precisa ser um valor válido do enum.
- `mileage` também é opcional: ausente (ou `null`) mantém; número inteiro válido grava (inclusive `0`). **Não há como voltar para `null` pelo PUT** — só sobrescrever com outro valor.
- `customer_id` e `customer_vehicle_id` também são opcionais: ausente (ou `null`) mantém o vínculo; ver [Vínculo com o cadastro de clientes](#vínculo-com-o-cadastro-de-clientes).
- Um PUT só com `title`/`plate`/`customer` (como o app envia) não apaga CPF/CEP/WhatsApp, a prioridade, a quilometragem nem os vínculos com cliente.
- 200 com o veículo atualizado. 400 em dado inválido (`CPF inválido`, `CEP inválido`, `WhatsApp inválido`, `prioridade inválida`, `quilometragem inválida`); nada é gravado se algum campo for inválido.

### Vínculo com o cadastro de clientes

Um veículo em `/api/lists` é, na prática, um **atendimento** (uma passagem pela oficina). Ele pode apontar para o [cliente](#clientes-apicustomers--só-ownermanager) e para o [veículo do cliente](#veículos-do-cliente):

- `customer_id` / `customer_vehicle_id`: inteiros ou `null`. Aparecem para **todos os papéis** (são só ids, não dados pessoais); o mecânico não tem acesso aos endpoints de clientes.
- Aceitos no `POST /api/lists` e no `PUT /api/lists/:id` (só dono/gerente). Ausente ou `null` = não muda (no POST fica `null`). `0` desvincula.
- Precisam ser da **mesma oficina** do atendimento: senão 400 `{ "error": "cliente inválido" }` / `{ "error": "veículo do cliente inválido" }`. Enviando os dois, o veículo precisa ser do cliente: senão 400 `{ "error": "veículo não pertence ao cliente" }`. Nada é gravado em caso de erro.
- Com `customer_vehicle_id` e sem `customer_id`, o atendimento é vinculado ao dono do veículo (se houver).
- Preenchimento automático (POST e PUT, só quando o id vem no body e o campo ficou vazio): `plate` ← placa do veículo do cliente; `title` ← `fabricante modelo` do veículo do cliente; `customer` ← nome do cliente; `customer_cpf` / `customer_cep` / `customer_whatsapp` ← `cpf` / `cep` / `mobile` do cliente. O que foi digitado nunca é sobrescrito.
- `title` continua obrigatório **depois** do preenchimento: sem título e sem `customer_vehicle_id` → 400 `título é obrigatório`; com veículo sem fabricante/modelo também.

```json
{ "customer_vehicle_id": 30 }
```
→ cria `{ "title": "VW Gol 1.6", "plate": "ABC1D23", "customer": "Maria Souza", "customer_id": 12, "customer_vehicle_id": 30, ... }`.

### Quilometragem

`mileage`: quilometragem do veículo em km, inteiro ou `null` (não informada). Visível para **todos os papéis** (dono, gerente e mecânico), em todas as respostas que devolvem veículo. Só dono/gerente gravam (`POST /api/lists` e `PUT /api/lists/:id`).

- Aceita apenas **número JSON inteiro** de `0` a `2000000`.
- 400 `{ "error": "quilometragem inválida" }` para negativo, acima de `2000000`, decimal (`12.5`), notação científica (`1e3`), string (`"87500"`), booleano, objeto ou array.
- `null` (ou chave ausente) = não informar: no POST fica `null`; no PUT mantém o valor atual.

```json
{ "id": 7, "title": "Gol 2012", "mileage": 87500 }
{ "id": 8, "title": "Uno 2015", "mileage": null }
```

### PUT /api/lists/:id/priority
Body: `{ "priority": "alta" }`. Só `owner`/`manager`.

- Prioridade: `normal` | `alta` | `urgente` (default `normal`). Visível para todos os papéis.
- **Não reordena** o pátio: a ordem continua por `position` (reorder manual).
- 200 com o veículo atualizado (JSON conforme o papel).
- 400 `{ "error": "prioridade inválida" }` / `dados inválidos` (sem `priority`)
- 403 `{ "error": "apenas gerentes podem gerenciar veículos" }` (mecânico)
- 404 veículo inexistente ou de outra oficina

### Dados do cliente (LGPD)

`customer_cpf`, `customer_cep` e `customer_whatsapp` ficam gravados **só com dígitos**. A entrada aceita máscara.

| Campo | Aceita | Grava | Erro (400) |
| --- | --- | --- | --- |
| `customer_cpf` | `""` ou 11 dígitos com dígitos verificadores válidos; máscara com `.`, `-`, espaço (`123.456.789-09`). Sequências repetidas (`111.111.111-11`) são recusadas. | `12345678909` | `{ "error": "CPF inválido" }` |
| `customer_cep` | `""` ou 8 dígitos; `-`, `.`, espaço (`01310-100`) | `01310100` | `{ "error": "CEP inválido" }` |
| `customer_whatsapp` | `""` ou DDD + número (10 ou 11 dígitos); `+`, `()`, `-`, `.`, espaço; prefixo `+55`/`55` é removido quando sobram 10–11 dígitos. DDD não começa com 0. | `11987654321` | `{ "error": "WhatsApp inválido" }` |

**Visibilidade:** os três campos só aparecem no JSON para `owner`/`manager` da oficina do veículo. Para `editor` (mecânico) as chaves **não existem** na resposta, em todos os endpoints que devolvem veículo (`GET /api/lists`, `GET /api/lists/:id` e as mutações que devolvem o veículo). O mecânico continua vendo `customer` (nome).

Dono/gerente:

```json
{ "id": 7, "title": "Gol 2012", "plate": "ABC1D23", "customer": "Maria Souza",
  "customer_cpf": "12345678909", "customer_cep": "01310100", "customer_whatsapp": "11987654321" }
```

Mecânico (mesmo veículo):

```json
{ "id": 7, "title": "Gol 2012", "plate": "ABC1D23", "customer": "Maria Souza" }
```

### DELETE /api/lists/:id
Remove veículo e serviços. Só `owner`/`manager`.

### PUT /api/lists/reorder
Body: `{ "ids": [3,1,2] }`. Só `owner`/`manager` (mecânico → 403). 204.

- Aceita o pátio inteiro **ou um subconjunto** (ex.: só os não entregues exibidos no pátio). Os ids enviados passam a ocupar, na ordem pedida, as mesmas vagas que já ocupavam; veículos fora do body não mudam de lugar.
- Vale para qualquer veículo da oficina, independente de quem o criou.
- Ids de outra oficina, inexistentes ou repetidos são ignorados (sem erro).
- Após a chamada, o pátio fica numerado `0..n-1` sem empates (`position`).

Ordenação de `GET /api/lists`: `position` asc, desempate por `id` asc. Novo veículo entra com `position` = maior posição do pátio + 1 (fim da fila).

### PUT /api/lists/:id/checkup-video
Lembrete fixo "Vídeo de check-up feito?" até alguém marcar. Body: `{ "done": true }` ou `{ "done": false }`.

- Quem pode: `owner`/`manager` em qualquer veículo da oficina; `editor` só nos veículos em que está atribuído (mesma regra de serviços/notas). Sem acesso ou outra oficina → 404.
- `true`: grava `checkup_video_done_at` (agora, UTC) e `checkup_video_done_by` (usuário atual). Reenviar `true` **mantém** data e autor originais.
- `false`: limpa os dois (`null`).
- 200 com o veículo atualizado (JSON conforme o papel).
- 400 `dados inválidos` (sem `done` ou não booleano) / `id inválido`

```json
{ "id": 7, "checkup_video_done_at": "2026-11-04T14:30:00Z", "checkup_video_done_by": 3, "checkup_video_done_by_name": "Ana Silva" }
{ "id": 8, "checkup_video_done_at": null, "checkup_video_done_by": null, "checkup_video_done_by_name": null }
```

`checkup_video_done_by_name` = `first_name last_name` (trim); se vazio, `username` (igual a `author_name` das notas).

### PUT /api/lists/:id/status
Body: `{ "status": "aguardando_peca" }`.
- `editor`, `owner` e `manager`: qualquer status válido da lista acima (fluxo linear é só sugestão de UI).
- Ao entrar em `entregue`, a API grava `delivered_at` (UTC). Reenviar `entregue` mantém a data original; sair de `entregue` limpa `delivered_at` (`null`).

## Serviços

### POST /api/lists/:id/items
Body: `{ "text": "string" }`. 201.

### PUT /api/lists/:id/items/:itemId
Body: `{ "text": "string", "completed": true }` (substitui os dois campos).

### DELETE /api/lists/:id/items/:itemId
204.

### PUT /api/lists/:id/items/reorder
Body: `{ "ids": [11, 10] }`. 204.

Mecânico pode CRUD de serviços **nos veículos da esteira**.

## Atribuições (só owner/manager)

- GET `/api/lists/:id/assignments`
- POST `/api/lists/:id/assignments` body `{ "user_id": 5 }`
- DELETE `/api/lists/:id/assignments/:userId`

## Orçamento

- POST `/api/lists/:id/quotes` body `{ "text": "string" }`
- GET `/api/lists/:id/quotes`
- DELETE `/api/lists/:id/quotes/:quoteId`

## Pendências

- POST `/api/lists/:id/flags` body `{ "flag_type": "procurando_peca", "note": "" }`
- GET `/api/lists/:id/flags`
- PATCH `/api/lists/:id/flags/:flagId/resolve`

## Informações importantes

Histórico de observações livres do veículo, com data/hora e autor. Mesma regra de acesso de serviços/orçamentos/pendências: `owner`/`manager` em qualquer veículo da oficina; `editor` só nos veículos da esteira (atribuído). Sem acesso ou veículo de outra oficina → 404.

### GET /api/lists/:id/notes
200 com array das informações **ativas** (não excluídas), da mais recente para a mais antiga.

```json
[
  {
    "id": 12,
    "task_list_id": 7,
    "author_id": 3,
    "author_name": "Ana Silva",
    "text": "Cliente pediu para ligar antes de trocar a correia.\nPrefere WhatsApp.",
    "created_at": "2026-11-02T18:30:00Z"
  }
]
```

`author_name` = `first_name last_name` (trim); se vazio, `username` (e-mail).

### POST /api/lists/:id/notes
Body: `{ "text": "string" }`. Todos os papéis com acesso ao veículo (incluindo mecânico).

- O texto é salvo com trim (quebras de linha internas são mantidas).
- 201 com a informação criada (mesmo formato do GET).
- 400 `{ "error": "texto é obrigatório" }` (vazio ou só espaços)
- 400 `{ "error": "texto deve ter no máximo 1000 caracteres" }` (contagem em caracteres Unicode)
- 404 veículo não encontrado / sem acesso

### DELETE /api/lists/:id/notes/:noteId
Exclusão **lógica**: some das respostas, mas a linha fica no banco com `deleted_at` e `deleted_by` (não expostos no JSON).

- `owner`/`manager`: excluem qualquer informação do veículo.
- Autor (qualquer papel, inclusive mecânico): exclui as próprias.
- 204 excluída
- 403 `{ "error": "apenas gerentes ou o autor podem excluir esta informação" }`
- 404 `{ "error": "informação não encontrada" }` (já excluída, inexistente ou de outro veículo)

## Clientes (`/api/customers`) — só owner/manager

Cadastro permanente de clientes da oficina, alimentado pela importação do OnMotor (somente leitura lá; ver [Sincronização OnMotor](#sincronização-onmotor)) e por cadastros locais. **LGPD:** CPF/CNPJ/RG, telefones, e-mails, endereço e data de nascimento só trafegam nestes endpoints, que são exclusivos de `owner`/`manager`. Mecânico (`editor`) → 403 em todos; usuário sem oficina → 404; cliente de outra oficina → 404.

Documentos, telefones e CEP são gravados **só com dígitos** (a entrada aceita máscara). `source`: `local` (criado aqui) | `onmotor` (importado; `source_id` = id no OnMotor).

### Objeto cliente

```json
{
  "id": 12, "workspace_id": 1, "source": "onmotor", "source_id": "4521",
  "name": "Maria Souza", "person_type": "pf",
  "cpf": "12345678909", "cnpj": "", "rg": "12.345.678-9",
  "phone": "1133334444", "mobile": "11987654321",
  "email": "maria@exemplo.com", "email_secondary": "",
  "cep": "01310100", "street": "Av. Paulista", "number": "1000", "complement": "ap 1",
  "district": "Bela Vista", "city": "São Paulo", "uf": "SP",
  "birth_date": "1980-05-03", "notes": "",
  "created_at": "2026-11-09T19:00:00Z", "updated_at": "2026-11-09T19:00:00Z", "synced_at": "2026-11-09T19:00:00Z",
  "vehicles": [ /* ver abaixo */ ],
  "last_service_at": "2026-10-20T13:10:00Z"
}
```

| Campo | Regra (400 quando inválido) |
| --- | --- |
| `name` | obrigatório no POST; se enviado no PUT não pode ser vazio (`nome é obrigatório`) |
| `person_type` | `""` \| `pf` \| `pj` (aceita também `F`/`J`/`física`/`jurídica`) (`tipo de pessoa inválido`) |
| `cpf` | igual a `customer_cpf` dos veículos (`CPF inválido`) |
| `cnpj` | `""` ou 14 dígitos com dígitos verificadores; máscara `.`, `/`, `-` (`CNPJ inválido`) |
| `phone` / `mobile` | DDD + número (10–11 dígitos), mesmas regras do `customer_whatsapp` (`telefone inválido` / `celular inválido`) |
| `email` / `email_secondary` | `""` ou `algo@dominio.tld`; gravado em minúsculas (`e-mail inválido`) |
| `cep` | 8 dígitos (`CEP inválido`) |
| `uf` | `""` ou 2 letras; gravado em maiúsculas (`UF inválida`) |
| `birth_date` | `""`/`null` ou `AAAA-MM-DD`, `DD/MM/AAAA`; não pode ser futura nem antes de 1900 (`data de nascimento inválida`). Resposta sempre `AAAA-MM-DD` ou `null` |
| `rg`, `street`, `number`, `complement`, `district`, `city`, `notes` | texto livre (trim) |

`last_service_at`: o mais recente entre a abertura (`opened_at`) das OS importadas do cliente e a criação dos nossos atendimentos (`/api/lists`) vinculados a ele; `null` se nunca houve.

### GET /api/customers
Query: `search`, `page`, `limit` (mesmos limites de `/api/lists`: `page` mínimo 1; `limit` 1–100, fora disso 20).

`search` não diferencia maiúsculas e casa com: nome, e-mails, **placa** de qualquer veículo do cliente (ignora `-` e espaços) e — quando o termo não tem letras — CPF, CNPJ, telefone e celular comparando só os dígitos (`123.456.789-09`, `(11) 98765` funcionam).

Ordem: **atualizados mais recentemente primeiro** (`updated_at` desc, desempate `id` desc). Importações que não mudam nada no cadastro não alteram `updated_at`.

```json
{
  "customers": [
    {
      "id": 12, "name": "Maria Souza", "cpf": "12345678909", "mobile": "11987654321", "...": "demais campos do cliente",
      "vehicles": [
        { "id": 30, "plate": "ABC1D23", "manufacturer": "VW", "model": "Gol 1.6", "year_model": 2015, "last_mileage": 87500 }
      ],
      "last_service_at": "2026-10-20T13:10:00Z"
    }
  ],
  "page": 1, "limit": 20, "total": 1, "total_pages": 1
}
```

Na lista, `vehicles` é compacto (`id`, `plate`, `manufacturer`, `model`, `year_model`, `last_mileage`); sempre array (pode ser `[]`).

### GET /api/customers/:id
Cliente completo + `vehicles` com todos os campos do [veículo do cliente](#veículos-do-cliente) + `last_service_at`.

### POST /api/customers
Cria cliente **local** (`source = local`). Body com os campos do objeto (só `name` obrigatório). 201 com o cliente (`vehicles: []`).

- 409 `{ "error": "CPF já cadastrado em outro cliente", "existing_id": 12 }` quando o CPF já existe na oficina.

### PUT /api/customers/:id
Edita cliente local ou importado. **Chave ausente ou `null` mantém** o valor; chave presente grava (`""` limpa). Mesmas validações e mesmo 409 de CPF (trocar para o CPF de outro cliente). 200 com o cliente completo.

Cliente importado (`source = onmotor`) pode ser editado, mas a **próxima sincronização sobrescreve** os campos que vierem do OnMotor.

## Veículos do cliente

Cadastro permanente do carro do cliente (diferente do **atendimento** em `/api/lists`, que é uma passagem pela oficina). Só `owner`/`manager`; mecânico → 403; outra oficina → 404.

`plate` é normalizada: só letras e números, maiúsculas (`abc-1d23` → `ABC1D23`). Placa é **única por oficina** quando preenchida (veículos importados podem vir sem placa).

```json
{
  "id": 30, "workspace_id": 1, "customer_id": 12, "source": "onmotor", "source_id": "889",
  "plate": "ABC1D23", "manufacturer": "VW", "model": "Gol 1.6", "transmission": "Manual",
  "year_fab": 2014, "year_model": 2015, "fuel": "Flex", "engine": "1.6", "doors": 4,
  "color": "Prata", "chassis": "9BWAB45U0FT000000", "renavam": "00000000000",
  "last_mileage": 87500,
  "created_at": "...", "updated_at": "...", "synced_at": "..."
}
```

`customer_id` pode ser `null` (dono desconhecido na importação). `year_fab`, `year_model` (1900 até ano atual + 2), `doors` (0–10) e `last_mileage` (0–2000000) são inteiros ou `null`. `last_mileage` sobe automaticamente quando uma OS importada traz km maior.

### GET /api/customers/:id/vehicles
200 com array de veículos do cliente (por placa). Cliente inexistente/outra oficina → 404.

### POST /api/customers/:id/vehicles
Cria veículo **local** para o cliente. `plate` obrigatória (`placa é obrigatória`). O dono é o `:id` da rota (`customer_id` do body é ignorado). 201 com o veículo.

- 409 `{ "error": "placa já cadastrada em outro veículo", "existing_id": 30 }`
- 400 `ano inválido`, `número de portas inválido`, `quilometragem inválida`, `dados inválidos` (tipo errado, ex.: `"year_model": "2015"`)

### PUT /api/customer-vehicles/:id
Edita veículo local ou importado (a próxima sincronização pode sobrescrever o importado). Chave ausente ou `null` mantém. `customer_id` transfere o carro para outro cliente da mesma oficina (senão 400 `cliente inválido`). Se enviada, `plate` não pode ficar vazia. Mesmo 409 de placa.

### GET /api/customer-vehicles/lookup?plate=ABC-1D23
Reconhece um carro conhecido pela placa (aceita máscara). 200 `{ "vehicle": { ... }, "customer": { ... } }` (`customer` = `null` se o veículo não tem dono). 404 `veículo do cliente não encontrado`; 400 `placa é obrigatória`.

### GET /api/customer-vehicles/:id/history
Histórico do carro juntando (a) OS importadas do OnMotor e (b) nossos atendimentos (`/api/lists`) vinculados ao veículo (`customer_vehicle_id`). Ordem: `opened_at` mais recente primeiro; registros sem data no fim.

```json
{
  "records": [
    {
      "source": "rmaxx", "id": 57, "list_id": 57,
      "opened_at": "2026-11-09T13:00:00Z", "delivered_at": null,
      "mileage": 91000, "status": "em_andamento", "type": "",
      "summary": "Alinhamento; Balanceamento", "items": null, "total": null
    },
    {
      "source": "onmotor", "id": 812, "list_id": null,
      "opened_at": "2026-03-10T12:00:00Z", "delivered_at": "2026-03-12T20:00:00Z",
      "mileage": 80000, "status": "Entregue", "type": "Revisão",
      "summary": "Troca de óleo; Filtro de óleo",
      "items": [
        { "kind": "servico", "description": "Troca de óleo", "quantity": 1, "unit_price": 120, "total": 120 },
        { "kind": "peca", "description": "Filtro de óleo", "quantity": 1, "unit_price": 45.9, "total": 45.9 }
      ],
      "total": 165.9
    }
  ]
}
```

- `source`: `onmotor` (OS importada; `id` = id da OS aqui) | `rmaxx` (nosso atendimento; `id` = `list_id` = id do veículo em `/api/lists`).
- `opened_at`: abertura da OS / criação do atendimento. `status`: texto do OnMotor como veio, ou o status da esteira.
- `summary`: notas da OS ou, sem notas, até 3 descrições de itens; no nosso atendimento, até 3 serviços (`items` da esteira) ou o título. Máx. 200 caracteres.
- `items` só para `onmotor` (formato flexível, como importado); `null` para `rmaxx`.

## Tokens de integração (só owner)

Credencial do robô de sincronização (máquina a máquina). Só o **dono** cria, lista e revoga (gerente e mecânico → 403). A API guarda apenas o **hash SHA-256** do token; o texto aparece **uma única vez**, na resposta do POST. Guarde-o num cofre/variável de ambiente do robô.

### GET /api/integrations/tokens
200 `{ "tokens": [ { "id": 3, "workspace_id": 1, "name": "Robô OnMotor", "last_used_at": "2026-11-09T18:00:00Z", "created_by": 1, "created_at": "...", "revoked_at": null } ] }` — mais novos primeiro, inclusive revogados. Nunca traz o token nem o hash.

### POST /api/integrations/tokens
Body `{ "name": "Robô OnMotor" }` (obrigatório, até 100 caracteres). 201:

```json
{ "id": 3, "name": "Robô OnMotor", "token": "rmx_2r9Qk0...43 caracteres base64url", "created_at": "2026-11-09T18:00:00Z" }
```

O token tem o prefixo `rmx_` + 32 bytes aleatórios em base64url.

### DELETE /api/integrations/tokens/:id
Revoga (204). Idempotente; token de outra oficina/inexistente → 404. Token revogado passa a receber 401 na importação.

## Auditoria (owner/manager)

### GET /api/audit
Query: `page`, `limit`, `category`, `actor_id`, `date_from`, `date_to`.

```json
{
  "logs": [
    {
      "id": 1,
      "actor_id": 2,
      "actor_name": "ana",
      "category": "acesso",
      "action": "login_success",
      "description": "",
      "created_at": "2026-08-17T22:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "total_pages": 1
}
```

Categorias: `acesso`, `veiculo`, `servico`, `financeiro`, `membros`.

## Objeto veículo (TaskList)

Campos relevantes: `id`, `created_at`, `updated_at`, `title`, `plate`, `customer`, `customer_cpf`\*, `customer_cep`\*, `customer_whatsapp`\*, `customer_id`, `customer_vehicle_id`, `user_id`, `workspace_id`, `status`, `priority`, `mileage`, `position`, `delivered_at`, `checkup_video_done_at`, `checkup_video_done_by`, `checkup_video_done_by_name`, `items[]`, `assignments[]`, `notes[]`.

\* Só para `owner`/`manager` (ver [Dados do cliente](#dados-do-cliente-lgpd)); ausentes para o mecânico. Veículos antigos recebem `""` na migração.

`priority`: `normal` | `alta` | `urgente`; veículos antigos recebem `normal` na migração.

`customer_id` / `customer_vehicle_id`: id ou `null`; sempre presentes, visíveis para todos os papéis. Veículos antigos ficam `null`.

`mileage`: km (inteiro) ou `null`; sempre presente e visível para todos os papéis. Veículos antigos ficam `null` na migração (coluna nova, nullable). Ver [Quilometragem](#quilometragem).

`checkup_video_done_at` / `checkup_video_done_by` / `checkup_video_done_by_name`: sempre presentes; `null` enquanto o vídeo não foi marcado (inclusive veículos antigos). Visíveis para todos os papéis.

Exemplo completo — dono/gerente:

```json
{
  "id": 7, "title": "Gol 2012", "plate": "ABC1D23", "customer": "Maria Souza",
  "customer_cpf": "12345678909", "customer_cep": "01310100", "customer_whatsapp": "11987654321",
  "status": "em_andamento", "priority": "alta", "mileage": 87500, "position": 0, "delivered_at": null,
  "checkup_video_done_at": "2026-11-04T14:30:00Z", "checkup_video_done_by": 3, "checkup_video_done_by_name": "Ana Silva",
  "items": [], "assignments": [{ "id": 1, "task_list_id": 7, "user_id": 3, "assigned_by": 1, "assigned_at": "..." }], "notes": []
}
```

Mecânico atribuído (mesmo veículo): idêntico, **sem** `customer_cpf`, `customer_cep` e `customer_whatsapp`.

## Matriz de permissões do veículo

| Ação | owner (Dono) | manager (Gerente) | editor (Mecânico) |
| --- | --- | --- | --- |
| Ver veículo (`GET /api/lists`, `/:id`) | pátio inteiro | pátio inteiro | só atribuídos (outros → 404) |
| Ver `customer_cpf` / `customer_cep` / `customer_whatsapp` | sim | sim | **nunca** (chaves omitidas) |
| Ver `priority`, `mileage` e `checkup_video_*` | sim | sim | sim |
| Criar / editar ficha (`POST /api/lists`, `PUT /:id`) | sim | sim | 403 |
| `PUT /:id/priority` | sim | sim | 403 |
| `PUT /:id/checkup-video` | sim | sim | só atribuídos (outros → 404) |
| `PUT /:id/status` | sim | sim | só atribuídos |
| `PUT /api/lists/reorder`, `DELETE /:id` | sim | sim | 403 |
| Ver `customer_id` / `customer_vehicle_id` | sim | sim | sim (só ids) |

Veículo de outra oficina → 404 para todos.

`notes`: informações importantes **ativas** do veículo, da mais recente para a mais antiga, no mesmo formato de `GET /api/lists/:id/notes`. Vem em `GET /api/lists`, `GET /api/lists/:id`, `PUT /api/lists/:id` e `POST /api/lists` (`[]`). Campo aditivo: clientes devem tratar ausência (API antiga) como lista vazia.

`delivered_at`: data/hora da entrega (RFC3339, UTC) quando `status = entregue`; `null` nos demais status. Veículos já entregues antes do campo existir recebem `updated_at` como estimativa (backfill no boot).

```json
{ "id": 7, "status": "entregue", "delivered_at": "2026-10-27T18:45:00Z", "updated_at": "2026-10-27T18:45:00Z" }
{ "id": 8, "status": "em_andamento", "delivered_at": null, "updated_at": "2026-10-28T09:10:00Z" }
```

IDs GORM antigos (`ID`, `CreatedAt`) podem aparecer se o model ainda usar tags maiúsculas em algum campo; clientes devem decodificar de forma tolerante.

## Erros comuns

- 400 dados inválidos (`título é obrigatório`, `CPF inválido`, `CEP inválido`, `WhatsApp inválido`, `quilometragem inválida`, `nome é obrigatório`, `CNPJ inválido`, ...)
- 409 cadastro duplicado na oficina: `{ "error": "...", "existing_id": 12 }` (CPF de cliente, placa de veículo do cliente)
- 401 token ausente, inválido ou na blacklist
- 403 transição de status ou mutação de pátio negada ao mecânico; exclusão de informação importante de outro autor pelo mecânico
- 404 veículo/item não encontrado (também usado para esconder recurso de outro workspace)
