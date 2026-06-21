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

Modelo de dados: cada **lista** (`TaskList`) é um bloco pertencente a um usuário, contendo vários **itens** (`TaskItem`) marcáveis dentro dela. Não existe mais o conceito de "tarefa solta" — toda tarefa marcável vive dentro de uma lista.

### GET /api/lists
Lista os blocos do usuário autenticado, já com os itens de cada um embutidos, com busca e paginação opcionais via query string.

**Query params (todos opcionais):**
- `search` — filtra listas cujo título contenha o texto informado
- `page` — número da página (padrão: 1)
- `limit` — listas por página (padrão: 20, máximo: 100)

**Exemplo:** `GET /api/lists?search=compras&page=1&limit=20`

**Resposta `200 OK`:**
```json
{
  "lists": [ /* array de objetos TaskList, ver formato abaixo */ ],
  "page": 1,
  "limit": 20,
  "total": 4,
  "total_pages": 1
}
```

### POST /api/lists
Cria uma nova lista vazia (sem itens) para o usuário autenticado.

**Body:**
```json
{ "title": "string (obrigatório)" }
```

**Respostas:**
- `201 Created` — objeto `TaskList` criado, com `items: []`
- `400 Bad Request` — `{"error": "título é obrigatório"}` ou `{"error": "dados inválidos"}`

### GET /api/lists/:id
Busca uma lista específica (com seus itens) — só retorna se pertencer ao usuário autenticado.

**Respostas:**
- `200 OK` — objeto `TaskList`
- `404 Not Found` — `{"error": "lista não encontrada"}`

### PUT /api/lists/:id
Atualiza só o título de uma lista existente.

**Body:**
```json
{ "title": "string" }
```

**Respostas:**
- `200 OK` — objeto `TaskList` atualizado
- `404 Not Found` — `{"error": "lista não encontrada"}`

### DELETE /api/lists/:id
Remove uma lista e **todos os itens dentro dela**.

**Respostas:**
- `204 No Content` — sem corpo na resposta
- `404 Not Found` — `{"error": "lista não encontrada"}`

### POST /api/lists/:id/items
Adiciona um novo item (não concluído) dentro de uma lista do usuário autenticado.

**Body:**
```json
{ "text": "string (obrigatório)" }
```

**Respostas:**
- `201 Created` — objeto `TaskItem` criado
- `400 Bad Request` — `{"error": "texto do item é obrigatório"}`
- `404 Not Found` — `{"error": "lista não encontrada"}`

### PUT /api/lists/:id/items/:itemId
Atualiza o texto e/ou o status de conclusão de um item.

**Body:**
```json
{ "text": "string", "completed": true }
```

**Respostas:**
- `200 OK` — objeto `TaskItem` atualizado
- `404 Not Found` — `{"error": "lista não encontrada"}` ou `{"error": "item não encontrado"}`

### DELETE /api/lists/:id/items/:itemId
Remove um item específico de dentro de uma lista.

**Respostas:**
- `204 No Content` — sem corpo na resposta
- `404 Not Found` — `{"error": "lista não encontrada"}` ou `{"error": "item não encontrado"}`

---

### PUT /api/lists/reorder
Atualiza a ordem de exibição das listas do usuário autenticado.

**Body:**
```json
{ "ids": [3, 1, 2] }
```
O array deve conter os IDs de todas as listas que o usuário quer reordenar, na nova ordem desejada. A primeira posição (índice 0) vira a `position` 0, e assim por diante.

**Respostas:**
- `204 No Content` — sem corpo na resposta
- `400 Bad Request` — `{"error": "lista de ids vazia"}` ou `{"error": "dados inválidos"}`

### PUT /api/lists/:id/items/reorder
Atualiza a ordem dos itens dentro de uma lista específica.

**Body:**
```json
{ "ids": [11, 10] }
```

**Respostas:**
- `204 No Content` — sem corpo na resposta
- `400 Bad Request` — `{"error": "lista de ids vazia"}`
- `404 Not Found` — `{"error": "lista não encontrada"}`

---

## Formato do objeto TaskList

```json
{
  "ID": 1,
  "CreatedAt": "2026-06-18T10:00:00Z",
  "UpdatedAt": "2026-06-18T10:00:00Z",
  "DeletedAt": null,
  "title": "Compras da semana",
  "user_id": 1,
  "items": [
    {
      "ID": 10,
      "CreatedAt": "2026-06-18T10:05:00Z",
      "UpdatedAt": "2026-06-18T10:05:00Z",
      "DeletedAt": null,
      "text": "Leite",
      "completed": false,
      "task_list_id": 1
    }
  ]
}
```

Nota: os campos `ID`, `CreatedAt`, `UpdatedAt` e `DeletedAt` (tanto na lista quanto em cada item) vêm com a primeira letra maiúscula por serem herdados de `gorm.Model`, sem tag `json` customizada — diferente de `title`/`text`/`completed`/`user_id`/`task_list_id`, que usam tags em minúsculo definidas explicitamente no model.

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
- O `PUT /api/lists/:id/items/:itemId` substitui `text` e `completed` de uma vez (não é atualização parcial) — ao marcar/desmarcar um item, sempre reenviar o `text` atual junto.
- Listas e itens são ordenados pelo campo `position` (ordem definida pelo usuário, manipulável via os endpoints de reorder), não mais por data de criação.