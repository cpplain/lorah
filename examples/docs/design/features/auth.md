# Auth

## 1. Overview

### Purpose

Define how Taskflow authenticates users and authorizes API requests. Users register and log in with email and password. The API issues JWTs for stateless authentication. Role-based access control restricts operations based on project membership.

### Goals

- Email/password registration and login
- JWT-based authentication for all API requests
- Role-based authorization per project (owner, editor, viewer)
- Password hashing with bcrypt

### Non-Goals

- OAuth2 or social login (v2)
- Multi-factor authentication (v2)
- User profile management beyond email and password
- Account-level authorization filtering (see `tasks.md`)

---

## 2. Scope Fences

### Out of Scope

- Task or project CRUD (see `tasks.md`, `projects.md`)
- `ErrorResponse` shape (defined in [architecture.md](../architecture.md) §5)
- Test framework and helpers (defined in [testing.md](../testing.md))

### Do NOT Change

- `ErrorResponse` interface shape ([architecture.md](../architecture.md) §5)
- Health endpoint contract ([architecture.md](../architecture.md) §6)
- JWT library (`jsonwebtoken`)

---

## 3. Dependencies

### Prerequisites

- PostgreSQL database with `users` table (created by migration)
- `JWT_SECRET` environment variable set
- `bcrypt` package for password hashing

### Build Order

1. User migration (`users` table)
2. Registration endpoint (`POST /api/auth/register`)
3. Login endpoint (`POST /api/auth/login`)
4. `requireAuth` middleware
5. `requireRole` middleware
6. `GET /api/auth/me` endpoint

---

## 4. Configuration

### Environment Variables

| Variable        | Required | Description                           |
| --------------- | -------- | ------------------------------------- |
| `JWT_SECRET`    | Yes      | Secret for signing and verifying JWTs |
| `JWT_EXPIRY`    | No       | Token lifetime (default: `2h`)        |
| `BCRYPT_ROUNDS` | No       | bcrypt salt rounds (default: `12`)    |

### Acceptance Criteria

- Given `JWT_SECRET` is not set, when the app starts, then it exits with a config validation error naming the missing variable.
- Given `JWT_EXPIRY` is not set, when a token is issued, then it expires 2 hours after issuance.

---

## 5. User Model

### Schema

| Column       | Type         | Constraints                 |
| ------------ | ------------ | --------------------------- |
| `id`         | uuid         | Primary key, auto-generated |
| `email`      | varchar(255) | Unique, not null            |
| `password`   | varchar(255) | Not null (bcrypt hash)      |
| `created_at` | timestamp    | Not null, default now()     |
| `updated_at` | timestamp    | Not null, default now()     |

Passwords are stored as bcrypt hashes. Plaintext passwords never persist to disk or logs.

---

## 6. Auth Routes

### 6.1 POST /api/auth/register

Creates a new user account.

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Success Response:** `201` with the user's `id` and `email` (no password).

**Behavior:**

1. Validate email format and password length (minimum 8 characters)
2. Check email uniqueness
3. Hash password with bcrypt
4. Insert user record
5. Return user identity (no token — user must log in separately)

#### Acceptance Criteria

- Given a valid email and password, when `POST /api/auth/register`, then the user is created and response is `201` with `id` and `email`.
- Given an email that already exists, when `POST /api/auth/register`, then response is `409` with error `"email_taken"`.
- Given a password shorter than 8 characters, when `POST /api/auth/register`, then response is `400` with error `"validation_error"`.

#### Boundary Conditions

- Given an empty email string, response is `400` with error `"validation_error"`.
- Given a missing `password` field, response is `400` with error `"validation_error"`.
- Given an email with leading/trailing whitespace (`" user@example.com "`), the email is trimmed before validation and storage.
- Given an email with uppercase letters, it is lowercased before uniqueness check and storage.

### 6.2 POST /api/auth/login

Authenticates a user and issues a JWT.

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Success Response:** `200` with a JWT token:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Behavior:**

1. Look up user by email
2. Compare password against stored bcrypt hash
3. On match, issue a signed JWT containing `sub` (user id), `email`, and `exp`
4. Return the token

#### Acceptance Criteria

- Given valid credentials, when `POST /api/auth/login`, then response is `200` with a valid JWT.
- Given a valid email with wrong password, when `POST /api/auth/login`, then response is `401` with error `"invalid_credentials"`.
- Given a non-existent email, when `POST /api/auth/login`, then response is `401` with error `"invalid_credentials"`. The error message is identical to wrong-password to prevent email enumeration.

#### Boundary Conditions

- Given a missing `email` field, response is `400` with error `"validation_error"`.
- Given a missing `password` field, response is `400` with error `"validation_error"`.

### 6.3 GET /api/auth/me

Returns the authenticated user's identity. Protected by `requireAuth`.

**Success Response:** `200` with `id` and `email`.

#### Acceptance Criteria

- Given a valid JWT in the Authorization header, when `GET /api/auth/me`, then response is `200` with the user's `id` and `email`.
- Given no Authorization header, when `GET /api/auth/me`, then response is `401`.

---

## 7. JWT Token

### Claims

| Claim   | Value                          |
| ------- | ------------------------------ |
| `sub`   | User `id` (uuid)               |
| `email` | User email                     |
| `iat`   | Issued-at timestamp            |
| `exp`   | Expiration (default: iat + 2h) |

### Signing

Tokens are signed with HS256 using `JWT_SECRET`. Tokens are stateless — there is no server-side token store or revocation list.

### Acceptance Criteria

- Given a JWT signed with `JWT_SECRET`, when verified, then the claims are returned.
- Given a JWT signed with a different secret, when verified, then verification fails.
- Given a JWT past its `exp` time, when verified, then verification fails.

---

## 8. Auth Middleware

### 8.1 requireAuth

Extracts and verifies the JWT from the `Authorization: Bearer <token>` header. On success, the decoded user identity is available to downstream handlers. On failure, returns `401` with error `"unauthorized"`.

#### Acceptance Criteria

- Given a valid Bearer token, when middleware runs, then user identity is available to the handler.
- Given no Authorization header, then response is `401`.
- Given an expired token, then response is `401`.
- Given a malformed token, then response is `401`.
- Given an Authorization header with a scheme other than Bearer (e.g., `Basic`), then response is `401`.

### 8.2 requireRole(...roles)

Factory function that returns middleware. Checks if the authenticated user has at least one of the specified roles for the target project (OR logic). Must be used after `requireAuth`.

Project membership and roles are looked up from the `project_members` table using the user's `id` and the `project_id` from the request parameters.

Returns `403` with error `"forbidden"` when the user lacks the required role.

#### Acceptance Criteria

- Given a user with role `owner` on a project, when the route requires `owner`, then the request proceeds.
- Given a user with role `viewer` on a project, when the route requires `editor`, then response is `403`.
- Given a user with role `editor`, when the route requires `editor` or `owner`, then the request proceeds (OR logic).
- Given a user who is not a member of the project, then response is `403`.

---

## 9. Error Catalog

| Status | `error`               | When                                           |
| ------ | --------------------- | ---------------------------------------------- |
| 400    | `validation_error`    | Missing or invalid fields in request body      |
| 401    | `unauthorized`        | No token, expired token, invalid token         |
| 401    | `invalid_credentials` | Wrong email/password on login                  |
| 403    | `forbidden`           | Insufficient role for the operation            |
| 409    | `email_taken`         | Registration with an email that already exists |

All errors use the `ErrorResponse` shape from [architecture.md](../architecture.md) §5.

---

## 10. Verification

### Test Commands

```bash
npm test -- --testPathPattern=auth
```

### Test Scenarios

1. **Register success**: valid email and password creates user, returns 201.
2. **Register duplicate email**: second registration with same email returns 409.
3. **Register invalid password**: password under 8 characters returns 400.
4. **Login success**: valid credentials return 200 with JWT.
5. **Login wrong password**: returns 401 with `invalid_credentials`.
6. **Login non-existent email**: returns 401 with `invalid_credentials` (same error as wrong password).
7. **Token verification**: valid token passes `requireAuth`; expired token returns 401.
8. **Role check**: user with `editor` role passes `requireRole("editor")`; `viewer` is rejected with 403.
9. **Me endpoint**: returns authenticated user's identity.

---

## 11. Related Specifications

- [../architecture.md](../architecture.md) §5 — `ErrorResponse` shape
- [../architecture.md](../architecture.md) §8 — glossary (member, project roles)
- [../testing.md](../testing.md) §4 — `createAuthenticatedAgent()`, `createTestUser()`
