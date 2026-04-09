# Guide: Organizing Design Specs

Design specs live in a `docs/design/` directory at the project root. This is the single location agents read to understand target behavior — the [spec-driven workflow](workflow.md) depends on it. For how to write individual specs, see the [specs guide](specs.md).

```
docs/design/
├── README.md           # navigational index
├── architecture.md     # foundation: scope, principles, tech stack
├── testing.md          # foundation: test strategy and patterns
├── ...                 # other foundation specs
├── glossary.md         # optional: project-specific terms
└── features/
    ├── auth.md         # feature: authentication flow
    ├── crud.md         # feature: create, read, update, delete
    └── ...             # other feature specs
```

Specs are divided into two categories: foundation specs at the top level, and feature specs in `features/`.

## Foundation specs

Foundation specs define cross-cutting concerns — conventions, constraints, and decisions that all feature specs assume. They live at the top level of `docs/design/`.

Examples:

- **Architecture** — scope, design principles, tech stack, repo structure.
- **Testing** — test framework, mocking patterns, test organization.
- **CLI conventions** — output formats, flag parsing, error handling.
- **Config** — config resolution, schema, environment variables.
- **Deployment** — CI/CD, containerization, secrets management.

Not every project needs all of these. The list depends on the project's cross-cutting concerns. The test is whether multiple feature specs would reference or assume the decision — if so, it belongs in a foundation spec.

## Feature specs

Feature specs are vertical, behavioral specs covering one user-facing capability end-to-end. They live in `docs/design/features/`.

Each feature spec covers behavior, inputs, outputs, error cases, and acceptance criteria for a single feature. Feature specs assume the reader has already read the foundation specs — they reference foundational decisions rather than repeating them.

Examples:

- **Auth** — login flow, session management, credential storage.
- **CRUD** — query, get, create, update, delete operations.
- **Plugins** — plugin structure, discovery, version pinning.

## README index

The `docs/design/README.md` is the navigational entry point. It groups specs into sections with a short description for each. Agents use it to discover which specs exist and where to start reading.

```markdown
# <Project> — Design Specifications

## Foundation

Read these first. They define conventions and cross-cutting concerns
that all feature specs assume.

| Spec                               | Description                          |
| ---------------------------------- | ------------------------------------ |
| [architecture.md](architecture.md) | Scope, design principles, tech stack |
| [testing.md](testing.md)           | Test strategy, mocking patterns      |
| ...                                | ...                                  |

## Feature Specs

Each spec covers one feature end-to-end: behavior, acceptance criteria,
error cases.

| Spec                                 | Description                    |
| ------------------------------------ | ------------------------------ |
| [features/auth.md](features/auth.md) | Login flow, session management |
| [features/crud.md](features/crud.md) | Table CRUD operations          |
| ...                                  | ...                            |
```

The "Read these first" framing signals the dependency: foundation specs are prerequisites for feature specs.

The README is a table of contents, not a summary. Descriptions are one line — enough to know whether to open the file, not enough to skip reading it.

## Glossary

The glossary defines project-specific terms used across specs. Place it as a separate `glossary.md` when the term list is large enough to clutter the README, or as a section in the README for smaller projects.

The [specs guide](specs.md) covers the "Defined vocabulary" property — terms defined in the glossary are used consistently across all specs. Agents treat synonyms as distinct concepts, so a single authoritative definition matters.

## Relationship to the workflow

In the [spec-driven workflow](workflow.md), the planning agent navigates from the README index to the relevant foundation and feature specs to understand what needs to be built. The quality of the design directory — clear categorization, accurate descriptions, complete specs — directly determines the quality of agent output.

## Example

A complete working example of this directory structure is in [`examples/docs/design/`](../../examples/docs/design/).
