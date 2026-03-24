---
title: backd Documentation
toc: false
---

# backd

A self-hosted Backend-as-a-Service (BaaS) for internal applications.

backd provides development teams with isolated PostgreSQL databases per application, auto-generated REST APIs with row-level access control, Deno-based serverless functions, and application-scoped user authentication — all driven by a GitOps-style configuration directory.

{{< cards >}}
  {{< card link="getting-started" title="Getting Started" subtitle="Install backd and create your first application in minutes." icon="play" >}}
  {{< card link="guides" title="Guides" subtitle="Learn how to work with apps, auth, data, functions, and storage." icon="book-open" >}}
  {{< card link="concepts" title="Concepts" subtitle="Understand the architecture, project layout, and design decisions." icon="light-bulb" >}}
  {{< card link="services" title="Services" subtitle="API, Functions, Worker — what each service does and how to run them." icon="server" >}}
  {{< card link="sdk" title="SDKs" subtitle="TypeScript and Go client libraries for browser and server use." icon="code" >}}
  {{< card link="reference" title="Reference" subtitle="Configuration schema, API reference, error codes, and environment variables." icon="document-text" >}}
{{< /cards >}}

## Key Features

- **Isolated databases** — each application gets its own PostgreSQL database
- **Auto-generated REST API** — full CRUD for every table, with filtering, pagination, and sorting
- **Row-level security** — CEL-based policies control who can read and write data
- **Deno functions** — TypeScript serverless functions with full database access
- **Background jobs** — enqueue and schedule work with retry and cron support
- **File storage** — S3-compatible uploads with automatic resolution in API responses
- **Shared auth domains** — multiple apps can share a single user pool
- **GitOps configuration** — everything defined in version-controlled YAML files
- **CLI tooling** — scaffold apps, validate configs, and manage secrets from the terminal
