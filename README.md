# todolist-api

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)
![Gin](https://img.shields.io/badge/Gin-Web%20Framework-008ECF)
![GORM](https://img.shields.io/badge/GORM-ORM-blue)
![JWT](https://img.shields.io/badge/Auth-JWT-orange)

API REST para gerenciamento de tarefas (To-Do List), construída em Go com autenticação JWT e arquitetura em camadas. Projeto de estudo prático da linguagem, pensado desde o início para servir de backend a um futuro app iOS em SwiftUI.

## Tecnologias

- **Go** — linguagem da API
- **Gin** — framework web / roteamento HTTP
- **GORM** + **SQLite** — ORM e banco de dados
- **golang-jwt** — geração e validação de tokens JWT
- **bcrypt** — hash de senhas
- **godotenv** — carregamento de variáveis de ambiente

## Arquitetura

O projeto segue uma arquitetura em três camadas, com responsabilidades bem separadas:

```
handler  →  service  →  repository
(HTTP)      (regras)     (banco de dados)
```

- **Handler**: só lida com a requisição/resposta HTTP — recebe o JSON, chama o service, devolve o status code certo. Não sabe nada sobre banco de dados.
- **Service**: contém a regra de negócio (validações, hash de senha, geração de token, garantir que um usuário só acesse suas próprias tarefas). Não sabe nada sobre HTTP.
- **Repository**: a única camada que conversa com o banco via GORM. Se o banco trocar de SQLite para PostgreSQL no futuro, só essa camada muda.

## Estrutura de pastas

```
todolist-api/
├── main.go               # monta as dependências e inicia o servidor
├── go.mod
├── go.sum
├── .env                   # não versionado — contém o JWT_SECRET
├── .gitignore
├── requests.http          # coleção de requisições para teste (REST Client)
├── API_REFERENCE.md       # documentação detalhada de cada endpoint
└── internal/
    ├── database/
    │   └── database.go    # conexão com o SQLite e migrations
    ├── models/
    │   ├── user.go
    │   └── task.go
    ├── repository/
    │   ├── user_repository.go
    │   └── task_repository.go
    ├── services/
    │   ├── auth_service.go
    │   └── task_service.go
    ├── handlers/
    │   ├── home_handler.go
    │   ├── auth_handler.go
    │   └── task_handler.go
    ├── middleware/
    │   └── auth_middleware.go  # valida o token JWT nas rotas protegidas
    ├── routes/
    │   └── routes.go
    └── utils/
        ├── jwt.go          # geração e validação do token
        └── response.go     # padroniza as respostas de erro
```

## Como rodar localmente

### Pré-requisitos
- Go 1.21 ou superior instalado ([go.dev/dl](https://go.dev/dl/))

### Passo a passo

```bash
# 1. Clonar o repositório
git clone git@github.com:RenanAlvesBCC/todolist-api.git
cd todolist-api

# 2. Criar o arquivo de variáveis de ambiente
echo "JWT_SECRET=troque-isso-por-uma-string-bem-longa-e-aleatoria" > .env

# 3. Baixar as dependências
go mod tidy

# 4. Rodar o servidor
go run main.go
```

Se tudo certo, o servidor sobe em `http://localhost:8080` e um arquivo `app.db` (SQLite) é criado automaticamente na raiz do projeto.

## Endpoints da API

| Método | Rota | Protegida? | Descrição |
|---|---|---|---|
| POST | `/register` | Não | Cria um novo usuário |
| POST | `/login` | Não | Autentica e devolve um token JWT |
| GET | `/api/tasks` | Sim | Lista as tarefas do usuário (com filtros e paginação) |
| POST | `/api/tasks` | Sim | Cria uma nova tarefa |
| GET | `/api/tasks/:id` | Sim | Busca uma tarefa específica |
| PUT | `/api/tasks/:id` | Sim | Atualiza uma tarefa |
| DELETE | `/api/tasks/:id` | Sim | Remove uma tarefa |

Rotas protegidas exigem o header `Authorization: Bearer <token>`, obtido no `/login`. Documentação completa de cada endpoint (formato de request/response, códigos de erro) em [`API_REFERENCE.md`](./API_REFERENCE.md).

## Testando

O arquivo `requests.http`, na raiz do projeto, contém uma coleção de requisições prontas para testar todos os endpoints usando a extensão [REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client) no VS Code — basta clicar em "Send Request" acima de cada bloco.

## Roadmap

- [x] Estrutura em camadas (handler / service / repository)
- [x] Cadastro e login com JWT
- [x] CRUD de tarefas vinculado ao usuário autenticado
- [x] Filtros, paginação e tratamento de erros consistente
- [ ] Apps consumindo essa API
- [ ] Testes automatizados
- [ ] Migração para PostgreSQL
- [ ] Deploy