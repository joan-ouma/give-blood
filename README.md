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