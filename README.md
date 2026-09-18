# Financial App Backend

API REST em Go para o Financial App. O projeto usa MVC, PostgreSQL como fonte
oficial dos dados e Redis somente para cache derivado de saldos e relatorios.

## Stack

- Go 1.24 e `net/http`
- PostgreSQL 17 com `pgx`
- Redis 8 com `go-redis`
- JWT HS256 e senhas com bcrypt
- Docker Compose para ambiente local

## Executar

```bash
docker compose up --build
```

A API fica em `http://localhost:8080`, o PostgreSQL em `localhost:5432` e o
Redis em `localhost:6379`. A migration e aplicada automaticamente pelo servico
`migrate`.

## Fluxo inicial

1. A inicializacao garante o administrador definido por `ADMIN_NAME`,
   `ADMIN_EMAIL` e `ADMIN_PASSWORD`.
2. `POST /api/v1/auth/login` autentica o usuario e define o cookie
   `financial_session` HttpOnly. Nao existe cadastro publico.
3. Use o cookie ou envie `Authorization: Bearer <token>` nas rotas protegidas.
4. Somente administradores criam usuarios e contas financeiras pelas rotas
   `/api/v1/admin/*`.
5. Use valores monetarios como centavos inteiros e datas como `YYYY-MM-DD`.
6. Em transferencias e pagamentos envie `Idempotency-Key`.

## Estrutura MVC

- `internal/model`: entidades, DTOs e validacoes de dominio.
- `internal/controller`: entrada HTTP e apresentacao JSON.
- `internal/service`: casos de uso e transacoes financeiras.
- `internal/repository`: persistencia PostgreSQL.
- `internal/cache`: cache Redis, sempre descartavel.
- `internal/router`: rotas e middlewares.

## Verificacoes

```bash
go test ./...
go vet ./...
```

O frontend usa o proxy `/api/backend` do Next.js para acessar esta API na
mesma origem e transportar o cookie HttpOnly.

## Recursos HTTP

Todas as rotas abaixo usam o prefixo `/api/v1`, exceto `GET /health`.

| Recurso | Rotas principais |
| --- | --- |
| Identidade | `POST /auth/login`, `POST /auth/logout`, `GET/PATCH /me` |
| Administracao | `GET/POST /admin/users`, `POST /admin/users/{userId}/accounts` |
| Contas | `GET /accounts`, `GET/PATCH/DELETE /accounts/{id}`, `GET /balances` |
| Categorias | `GET/POST /categories`, `PATCH/DELETE /categories/{id}` |
| Extrato | `GET /transactions`, `POST /incomes`, `POST /expenses`, atualizacao e estorno |
| Transferencias | `GET/POST /transfers`, `POST /transfers/{id}/reversal` |
| Dividas | `GET/POST /debts`, detalhe, atualizacao e exclusao condicionada |
| Pagamentos | `GET/POST /debts/{id}/payments`, `POST /payments/{id}/reversal` |
| Planejamento | CRUD em `/planned-cash-flows` e realizacao atomica |
| Relatorios | gastos, media mensal, projecoes e exportacao CSV |

Contas e categorias com historico sao arquivadas pelas rotas `DELETE`. Uma
divida so pode ser removida se ainda nao possuir pagamentos; caso contrario,
deve ser cancelada por `PATCH`. Lancamentos efetivados sao corrigidos por
estorno, preservando o historico.
