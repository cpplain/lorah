# Taskflow — Design Specifications

Design documentation for Taskflow, a task management API.

---

## Foundation

Read these first. They define conventions and cross-cutting concerns
that all feature specs assume.

| Spec                               | Description                             |
| ---------------------------------- | --------------------------------------- |
| [architecture.md](architecture.md) | Scope, design principles, tech stack    |
| [testing.md](testing.md)           | Test strategy, framework, test patterns |

## Feature Specs

Each spec covers one feature end-to-end: behavior, acceptance criteria,
error cases.

| Spec                                 | Description                                   |
| ------------------------------------ | --------------------------------------------- |
| [features/auth.md](features/auth.md) | JWT authentication, session management        |
| features/tasks.md                    | Task CRUD, status lifecycle, assignment (TBD) |
| features/projects.md                 | Project management, membership, roles (TBD)   |
