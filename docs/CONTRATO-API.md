# Contrato da API

Base: `http://localhost:8080` (dev) ou a URL de produção no Render.

Autenticação: `Authorization: Bearer <jwt>` nas rotas `/api`. Token expira em 24h. Logout coloca o token na blacklist.

No JSON, `lists` = veículos e `items` = serviços. Ver [GLOSSARIO.md](./GLOSSARIO.md).

## Públicos

### POST /register
Body: `{ "username": "string", "password": "string" }`
- 201 `{ "message": "usuário criado com sucesso" }`
- 409 usuário já existe

### POST /login
Body: `{ "username": "string", "password": "string" }`
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
Lista membros.

### DELETE /api/workspace/members/:userId
Remove membro (dono).

## Veículos (`/api/lists`)

### GET /api/lists
Query: `search`, `page`, `limit`, `status`, `mine=true`.

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

Status: `em_andamento` | `aguardando_orcamento` | `aguardando_peca` | `aprovado` | `concluido`.

### POST /api/lists
Cria veículo. Só `owner`/`manager`. Body: `{ "title": "string" }`. 201.

### GET /api/lists/:id
Detalhe. Mecânico só se atribuído ou membro com acesso ao veículo da esteira.

### PUT /api/lists/:id
Título. Só `owner`/`manager`.

### DELETE /api/lists/:id
Remove veículo e serviços. Só `owner`/`manager`.

### PUT /api/lists/reorder
Body: `{ "ids": [3,1,2] }`. Só `owner`/`manager`. 204.

### PUT /api/lists/:id/status
Body: `{ "status": "aguardando_peca" }`.
- `editor`: só entre `em_andamento`, `aguardando_orcamento`, `aguardando_peca`.
- `owner`/`manager`: qualquer transição.

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

Campos relevantes: `id`, `created_at`, `updated_at`, `title`, `user_id`, `workspace_id`, `status`, `position`, `items[]`, `assignments[]`.

IDs GORM antigos (`ID`, `CreatedAt`) podem aparecer se o model ainda usar tags maiúsculas em algum campo; clientes devem decodificar de forma tolerante.

## Erros comuns

- 401 token ausente, inválido ou na blacklist
- 403 transição de status ou mutação de pátio negada ao mecânico
- 404 veículo/item não encontrado (também usado para esconder recurso de outro workspace)
