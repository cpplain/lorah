# Testing

---

## 1. Overview

### Purpose

Define the test strategy for Taskflow. Tests verify API behavior, business logic, and database interactions against a real PostgreSQL instance.

### Goals

- Every API endpoint tested with Supertest against a real PostgreSQL instance
- Co-located tests: test files live next to the code they test in `__tests__/` directories
- Fast feedback: unit tests run without Docker, integration tests run against PostgreSQL

### Non-Goals

- End-to-end browser testing (no frontend)
- Performance or load testing
- Mutation testing

---

## 2. Framework

- **Jest** with `ts-jest` preset
- **Supertest** for HTTP endpoint testing
- Environment: `node`

### Configuration

Jest uses a multi-project configuration to separate unit and integration tests:

```javascript
// jest.config.js

const common = {
  preset: "ts-jest",
  testEnvironment: "node",
  roots: ["<rootDir>/src"],
};

module.exports = {
  testTimeout: 10000,
  projects: [
    {
      ...common,
      displayName: "unit",
      testMatch: ["**/__tests__/**/*.test.ts"],
      testPathIgnorePatterns: ["\\.integration\\.test\\.ts$"],
    },
    {
      ...common,
      displayName: "integration",
      testMatch: ["**/__tests__/**/*.integration.test.ts"],
      setupFilesAfterEnv: ["<rootDir>/src/__tests__/setup.ts"],
    },
  ],
};
```

### File Naming

- **Unit tests** (`*.test.ts`): no database required. Run without Docker.
- **Integration tests** (`*.integration.test.ts`): require a running PostgreSQL instance. The global setup file connects to the test database and runs migrations.

Tests run in parallel by default. PostgreSQL handles concurrent access.

---

## 3. Setup/Teardown

Tests use a dedicated PostgreSQL test database (configured via `DB_TEST_NAME` env var) to avoid interfering with development data.

```typescript
// src/__tests__/setup.ts

beforeAll(async () => {
  await db.initialize(); // connect and run migrations on test database
});

afterEach(async () => {
  await cleanup(); // truncate all tables between tests
});

afterAll(async () => {
  await db.close(); // destroy connection pool
});
```

---

## 4. Test Helpers

Factory functions create consistent fixture data:

```typescript
// src/__tests__/helpers.ts

async function createTestUser(overrides?): Promise<User>;
async function createTestProject(overrides?): Promise<Project>;
async function createTestTask(overrides?): Promise<Task>;
async function createAuthenticatedAgent(role?: string): Promise<SupertestAgent>;
async function cleanup(): Promise<void>; // truncates all tables in dependency order
```

Each factory returns a record with sensible defaults that can be overridden. `cleanup()` truncates tables in reverse foreign-key order. `createAuthenticatedAgent()` returns a Supertest agent with a valid JWT, bypassing the login flow.

---

## 5. Running Tests

| Command                    | Description                            |
| -------------------------- | -------------------------------------- |
| `npm test`                 | Run all tests (unit + integration)     |
| `npm run test:unit`        | Run unit tests only (no Docker needed) |
| `npm run test:integration` | Run integration tests only             |
| `npm run test:watch`       | Watch mode for all tests               |

---

## 6. Coverage Goals

| Layer       | Framework | Target             | Execution Time |
| ----------- | --------- | ------------------ | -------------- |
| Unit        | Jest      | 80%+ code coverage | < 10 seconds   |
| Integration | Supertest | 100% API endpoints | < 30 seconds   |

---

## 7. Related Specifications

- [architecture.md](architecture.md) — tech stack, repo structure, co-location principle
- [features/auth.md](features/auth.md) — `createAuthenticatedAgent()` usage in auth tests
