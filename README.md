# Blood MVP API

This is the Go backend service for the Blood Donation MVP.

## Requirements
- Docker and Docker Compose
- Go 1.26.2 (optional, for running/testing locally without Docker)

## Configuration
All environment variables are located in `.env` (copied from `.env.example`).
Key environment variables:
- `PORT`: Port the API listents on (e.g. `8080`).
- `MONGO_URI`: MongoDB connection string.
- `JWT_SECRET`: Signature key for HMAC-SHA256 JWT tokens.
- `ALLOWED_ORIGIN`: Allowed origins for CORS policy.

## Running the Application
To spin up the service (including MongoDB and Go with hot-reloading using `air`):
```bash
docker compose up --build
```

The API will be available at `http://localhost:8080`.

## API Endpoints (Auth Section)
- `POST /api/auth/register` - Create user. Request payload: `{ "email", "password", "role", "name" }`
- `POST /api/auth/login` - Login. Request payload: `{ "email", "password" }`. Returns access token in JSON body and sets `refreshToken` httpOnly cookie.
- `POST /api/auth/refresh` - Refresh access token using httpOnly cookie. Returns new access token.
- `GET /api/auth/me` - Retrieve current user details. Requires `Authorization: Bearer <token>` header.

## Day 1 Done

### Implemented:
- Go project structured layout (`/cmd/api/main.go`, `/internal/config`, `/internal/auth`, `/internal/user`, `/internal/db`, `/internal/httpserver`).
- Environment variable validation and error handling on boot.
- MongoDB connection pooling with fast failover timeouts.
- Idempotent index creation on boot (unique index on `users.email`).
- Safe password storage using Bcrypt (cost 12).
- JWT issuing and validation (15-minute expiration) with Bearer token authentication middleware injecting details into context.
- Refresh token handling (7 days, Secure, httpOnly, SameSite=Strict cookie).
- User validation on registration (email format, 8+ character password, role exactly "agency" or "donor").
- In-memory rate limiting on login endpoint (5 attempts per IP+email per 15 minutes).
- Fully dockerized environment with Hot Reload (`air`) and database healthcheck sequencing.

### Explicitly Deferred:
- Refresh token rotation / tracking table (access tokens are refreshed using valid cryptographic signatures; reuse protection table is a fast-follow for Day 2).
