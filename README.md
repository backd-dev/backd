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
