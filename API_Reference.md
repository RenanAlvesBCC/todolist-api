# API Reference — todolist-api

Backend em Go (Gin + GORM + SQLite) com autenticação JWT. Este documento descreve todos os endpoints disponíveis, pensado tanto como referência durante o desenvolvimento do backend quanto como contrato para o futuro app em SwiftUI.

**Base URL (desenvolvimento local):** `http://localhost:8080`

## Autenticação

O fluxo é: o cliente chama `/register` uma vez, depois `/login` para receber um token JWT. Esse token deve ser enviado no header `Authorization` em todas as rotas protegidas (prefixo `/api`), no formato:

```
Authorization: Bearer <token>
```

O token expira 24 horas após a geração. Não existe (ainda) endpoint de refresh — quando expira, o usuário precisa logar de novo.

---

## Endpoints públicos

### POST /register
Cria um novo usuário.

**Body:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Respostas:**
- `201 Created` — `{"message": "usuário criado com sucesso"}`
- `400 Bad Request` — `{"error": "dados inválidos"}` (campos ausentes)
- `409 Conflict` — `{"error": "usuário já existe"}`

### POST /login
Autentica e devolve um token JWT.

**Body:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Respostas:**
- `200 OK` — `{"token": "<jwt>"}`
- `400 Bad Request` — `{"error": "dados inválidos"}`
- `401 Unauthorized` — `{"error": "usuário ou senha incorretos"}`

---

## Endpoints protegidos (exigem `Authorization: Bearer <token>`)

### GET /api/tasks
Lista as tarefas do usuário autenticado, com filtros e paginação opcionais via query string.

**Query params (todos opcionais):**
- `completed` — `true` ou `false`, filtra por status
- `search` — filtra tarefas cujo título contenha o texto informado
- `page` — número da página (padrão: 1)
- `limit` — itens por página (padrão: 10, máximo: 100)

**Exemplo:** `GET /api/tasks?completed=false&search=estudar&page=1&limit=10`

**Resposta `200 OK`:**
```json
{
  "tasks": [ /* array de objetos Task, ver formato abaixo */ ],
  "page": 1,
  "limit": 10,
  "total": 4,
  "total_pages": 1
}
```

### POST /api/tasks
Cria uma nova tarefa para o usuário autenticado.

**Body:**
```json
{
  "title": "string (obrigatório)",
  "description": "string (opcional)"
}
```

**Respostas:**
- `201 Created` — objeto `Task` criado
- `400 Bad Request` — `{"error": "título é obrigatório"}` ou `{"error": "dados inválidos"}`

### GET /api/tasks/:id
Busca uma tarefa específica — só retorna se ela pertencer ao usuário autenticado.

**Respostas:**
- `200 OK` — objeto `Task`
- `404 Not Found` — `{"error": "tarefa não encontrada"}`

### PUT /api/tasks/:id
Atualiza uma tarefa existente do usuário autenticado.

**Body:**
```json
{
  "title": "string",
  "description": "string",
  "completed": true
}
```

**Respostas:**
- `200 OK` — objeto `Task` atualizado
- `404 Not Found` — `{"error": "tarefa não encontrada"}`

### DELETE /api/tasks/:id
Remove uma tarefa do usuário autenticado.

**Respostas:**
- `204 No Content` — sem corpo na resposta
- `404 Not Found` — `{"error": "tarefa não encontrada"}`

---

## Formato do objeto Task

```json
{
  "ID": 1,
  "CreatedAt": "2026-06-18T10:00:00Z",
  "UpdatedAt": "2026-06-18T10:00:00Z",
  "DeletedAt": null,
  "title": "Estudar Go",
  "description": "Terminar a fase 5 do projeto",
  "completed": false,
  "user_id": 1
}
```

Nota: os campos `ID`, `CreatedAt`, `UpdatedAt` e `DeletedAt` vêm com a primeira letra maiúscula porque são herdados de `gorm.Model` e ainda não têm uma tag `json` customizada — diferente de `title`, `description`, `completed` e `user_id`, que usam tags em minúsculo definidas no model. Vale padronizar isso numa fase futura de refinamento.

---

## Erros de autenticação (qualquer rota sob /api)

Esses erros vêm do middleware, antes mesmo do handler da rota ser executado:

- `401 Unauthorized` — `{"error": "token não fornecido"}` (header `Authorization` ausente)
- `401 Unauthorized` — `{"error": "formato de token inválido"}` (não está como `Bearer <token>`)
- `401 Unauthorized` — `{"error": "token inválido ou expirado"}`

---

## Observações para o app SwiftUI (uso futuro)

- Guardar o token recebido no login em local seguro (Keychain do iOS), nunca em `UserDefaults`.
- Incluir o token no header `Authorization` em toda chamada às rotas `/api/*`.
- Tratar respostas `401` redirecionando para a tela de login, já que o token expira em 24h e ainda não existe refresh automático.
- O campo `completed` no `PUT /api/tasks/:id` precisa ser enviado mesmo que não tenha mudado — o endpoint substitui os três campos (`title`, `description`, `completed`) de uma vez, não faz atualização parcial.