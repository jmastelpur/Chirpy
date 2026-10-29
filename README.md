# Chirpy

Chirpy is a small Twitter-style social API written in Go. Users sign up, log in, and post short messages called "chirps" (140 characters max). It uses PostgreSQL for storage, JWT access tokens with refresh tokens for auth, and a webhook that upgrades users to "Chirpy Red" membership.

## Features

- User registration and login with Argon2id password hashing
- JWT access tokens (1 hour) and refresh tokens (60 hours) that can be revoked
- Create, list, and delete chirps
- Filter chirps by author and sort by creation date
- Profanity filter that replaces `kerfuffle`, `sharbert`, and `fornax` with `****`
- Webhook endpoint for "Chirpy Red" upgrades, protected by an API key
- Static file server under `/app/` with a hit counter shown at `/admin/metrics`

## Requirements

- [Go](https://go.dev/dl/) 1.26+
- [PostgreSQL](https://www.postgresql.org/)
- [goose](https://github.com/pressly/goose) for database migrations
- [sqlc](https://sqlc.dev/) (only if you change the SQL queries)

## Setup

1. Clone the repository:

   ```bash
   git clone https://github.com/Jmastelpur/Chirpy.git
   cd Chirpy
   ```

2. Create a Postgres database:

   ```bash
   createdb chirpy
   ```

3. Create a `.env` file in the project root:

   ```env
   DB_URL=postgres://<user>:<password>@localhost:5432/chirpy?sslmode=disable
   PLATFORM=dev
   SECRET=<a long random string used to sign JWTs>
   POLKA_KEY=<API key expected from the Polka webhook>
   ```

   | Variable    | Description                                                              |
   |-------------|--------------------------------------------------------------------------|
   | `DB_URL`    | PostgreSQL connection string                                             |
   | `PLATFORM`  | Set to `dev` to allow `POST /admin/reset`; any other value disables it   |
   | `SECRET`    | Secret key for signing and validating JWTs                               |
   | `POLKA_KEY` | API key that the Polka payment webhook must send                         |

   You can generate a secret with `openssl rand -base64 64`.

4. Run the migrations:

   ```bash
   cd sql/schema
   goose postgres "$DB_URL" up
   cd ../..
   ```

5. Start the server:

   ```bash
   go build -o out && ./out
   ```

   The server listens on `http://localhost:8080`.

## Running tests

```bash
go test ./...
```

## Regenerating database code

Queries live in `sql/queries/` and the generated Go code lives in `internal/database/`. After editing a query, run:

```bash
sqlc generate
```

## API reference

Authenticated endpoints expect a header in the form `Authorization: Bearer <token>`. Errors come back as JSON:

```json
{ "error": "message" }
```

### Users

#### `POST /api/users`

Creates a user.

```json
{ "email": "user@example.com", "password": "hunter2" }
```

Response `201 Created`:

```json
{
  "id": "uuid",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z",
  "email": "user@example.com",
  "is_chirpy_red": false
}
```

#### `PUT /api/users`

Updates the email and password of the logged-in user. Requires an access token.

```json
{ "email": "new@example.com", "password": "newpassword" }
```

Response `200 OK`:

```json
{ "email": "new@example.com" }
```

### Authentication

#### `POST /api/login`

```json
{ "email": "user@example.com", "password": "hunter2" }
```

Response `200 OK`:

```json
{
  "id": "uuid",
  "created_at": "...",
  "updated_at": "...",
  "email": "user@example.com",
  "token": "<access JWT, valid for 1 hour>",
  "refresh_token": "<refresh token, valid for 60 hours>",
  "is_chirpy_red": false
}
```

#### `POST /api/refresh`

Sends the **refresh token** as the bearer token and gets a new access token back.

Response `200 OK`:

```json
{ "token": "<new access JWT>" }
```

#### `POST /api/revoke`

Sends the **refresh token** as the bearer token and revokes it. Response `204 No Content`.

### Chirps

#### `POST /api/chirps`

Creates a chirp. Requires an access token. The body can be at most 140 characters.

```json
{ "body": "Hello, Chirpy!" }
```

Response `201 Created`:

```json
{
  "id": "uuid",
  "created_at": "...",
  "updated_at": "...",
  "body": "Hello, Chirpy!",
  "user_id": "uuid"
}
```

#### `GET /api/chirps`

Lists chirps sorted by creation date, oldest first by default.

| Query parameter | Description                                  |
|-----------------|----------------------------------------------|
| `author_id`     | Only return chirps from this user ID         |
| `sort`          | `asc` (default) or `desc`                    |

Example: `GET /api/chirps?author_id=<uuid>&sort=desc`

#### `DELETE /api/chirps/{chirpID}`

Deletes a chirp. Requires an access token, and only the chirp's author can delete it.

| Status | Meaning                          |
|--------|----------------------------------|
| `204`  | Deleted                          |
| `403`  | You are not the author           |
| `404`  | Chirp not found                  |

### Webhooks

#### `POST /api/polka/webhooks`

Upgrades a user to Chirpy Red. Requires `Authorization: ApiKey <POLKA_KEY>`.

```json
{
  "event": "user.upgraded",
  "data": { "user_id": "uuid" }
}
```

Returns `204 No Content`. Any event other than `user.upgraded` is acknowledged and ignored. Returns `404` if the user doesn't exist.

### Admin

| Method | Path              | Description                                                   |
|--------|-------------------|---------------------------------------------------------------|
| `GET`  | `/admin/healthz`  | Health check                                                  |
| `GET`  | `/admin/metrics`  | HTML page showing how many times `/app/` has been visited     |
| `POST` | `/admin/reset`    | Deletes all users (and their chirps). Only works when `PLATFORM=dev` |

### Static files

`GET /app/*` serves files from the project root, such as `index.html` and `assets/logo.png`. Each request increments the metrics counter.

## Project structure

```
.
├── main.go                 # HTTP server and route handlers
├── index.html              # Static landing page served at /app/
├── assets/                 # Static assets
├── internal/
│   ├── auth/               # Password hashing, JWTs, refresh tokens, API keys
│   └── database/           # sqlc-generated query code
├── sql/
│   ├── schema/             # goose migrations
│   └── queries/            # SQL queries used by sqlc
└── sqlc.yaml               # sqlc configuration
```
