# backd

A self-hosted internal Backend-as-a-Service (BaaS) written in Go.

## Features

- Isolated PostgreSQL databases per application
- Auto-generated REST APIs with row-level access control
- Deno-based serverless functions
- GitOps-style configuration

## Quick Start

```bash
# Build the project
make build

# Start development stack
make docker-up

# Validate configuration
make validate
```

## Documentation

See the `_docs/` directory for detailed documentation.

## API Endpoints

### Authentication

```bash
# Register user
POST /v1/auth/your_app/local/register
{
  "username": "user@example.com",
  "password": "secure-password"
}

# Login
POST /v1/auth/your_app/local/login
{
  "username": "user@example.com", 
  "password": "secure-password"
}

# Refresh token
POST /v1/auth/your_app/refresh
{
  "token": "session-token"
}
```

### Data (CRUD)

```bash
# Create record
POST /v1/data/your_app/posts
{
  "title": "Hello World",
  "content": "My first post"
}

# List records
GET /v1/data/your_app/posts?limit=10&offset=0

# Get specific record
GET /v1/data/your_app/posts/123

# Update record
PUT /v1/data/your_app/posts/123
{
  "title": "Updated Title"
}
```

### Storage

```bash
# Upload file
POST /v1/storage/your_app/upload
Content-Type: multipart/form-data

# Get file info + URL
GET /v1/storage/your_app/files/file-id

# Delete file
DELETE /v1/storage/your_app/files/file-id
```

### Functions

```bash
# Call function
POST /v1/your_app/functions/my-function
{
  "param1": "value1"
}
```

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Check TypeScript SDK
make sdk-check
```

## License

TODO: Add license
