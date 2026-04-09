# Architecture

---

## 1. Scope

Taskflow is a task management API. The initial release is API-only — no web UI. The system exposes a RESTful API consumed by CLI tools and (eventually) a frontend. All feature specs design for API endpoints, not UI components.

---

## 2. Design Principles

- **Feature-based organization**: code is organized by domain feature, not technical layer.
- **Co-location**: tests, types, and related code live together within their feature module.
- **Barrel exports**: each module exports through `index.ts` for clean import paths.
- **Audit trail**: every mutation logs to `audit_log` with the acting user, timestamp, and change details.

---

## 3. Tech Stack

| Decision               | Rationale                                                                                                                 |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| **Node.js + Express**  | Lightweight, well-documented, large ecosystem. Express 5 for native async error handling.                                 |
| **PostgreSQL**         | Relational model fits task/project/user relationships. Supports concurrent access and standard tooling (Knex).            |
| **Knex.js**            | Query builder without ORM overhead. Provides migrations, connection pooling, and a fluent query API.                      |
| **JWT authentication** | Stateless auth tokens for API consumers. No server-side session store required. See [features/auth.md](features/auth.md). |
| **Winston logging**    | Structured JSON logs for aggregation. Severity levels: error, warn, info, debug.                                          |
| **Helmet**             | Standard HTTP security headers (HSTS, X-Content-Type-Options, X-Frame-Options, CSP).                                      |

---

## 4. Repo Structure

```
taskflow/
├── src/
│   ├── server.ts                 # Express app setup, middleware, listen
│   ├── auth/                     # Auth routes, middleware, token utilities
│   │   ├── routes.ts
│   │   ├── middleware.ts
│   │   ├── token.ts
│   │   └── __tests__/
│   ├── tasks/                    # Task CRUD routes, queries
│   │   ├── routes.ts
│   │   ├── queries.ts
│   │   └── __tests__/
│   ├── projects/                 # Project management routes, queries
│   │   ├── routes.ts
│   │   ├── queries.ts
│   │   └── __tests__/
│   ├── shared/
│   │   ├── db.ts                 # Database connection, initialize, close
│   │   ├── errors.ts             # Error classes, error handler middleware
│   │   └── logger.ts             # Winston logger configuration
│   └── database/
│       └── migrations/           # Knex migration files
├── package.json
├── knexfile.ts
├── tsconfig.json
└── docs/design/                  # This directory
```

---

## 5. Error Response Shape

All API error responses use a consistent shape:

```typescript
interface ErrorResponse {
  error: string; // machine-readable code (e.g., "not_found", "validation_error")
  error_description: string; // human-readable message
}
```

Feature specs define their own error catalogs (status codes and `error` values) but use this shape.

---

## 6. Health Endpoint

`GET /api/health` — used by container probes and monitoring. Not authenticated.

Returns `200` with `{ "status": "ok" }` when the server is accepting requests. No deep checks (database connectivity, external services) — the startup sequence gates on database availability before listening.

---

## 7. Known Limitations

- Single replica deployment — no horizontal scaling in v1. Acceptable for expected load.
- No rate limiting in v1 — add before public exposure.

---

## 8. Glossary

| Term          | Definition                                                                  |
| ------------- | --------------------------------------------------------------------------- |
| **task**      | A unit of work with a title, description, status, and optional assignee     |
| **project**   | A container for tasks with membership and role-based access                 |
| **member**    | A user associated with a project in a specific role (owner, editor, viewer) |
| **status**    | Task lifecycle state: `open`, `in_progress`, `done`, `archived`             |
| **assignee**  | The user responsible for completing a task                                  |
| **audit log** | Append-only record of mutations: user, timestamp, entity, action, diff      |

---

## 9. Related Specifications

- [testing.md](testing.md) — test strategy, framework, patterns
- [features/auth.md](features/auth.md) — JWT authentication, middleware
