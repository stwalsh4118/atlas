# PBI-1: Project Infrastructure Setup

[View in Backlog](../backlog.md#user-content-1)

## Overview

Set up the complete development environment for the Property Boundary Viewer project, including Docker containerization, PostgreSQL with PostGIS, development tooling, and project structure for both Go backend and frontend applications.

## Problem Statement

Before any development can begin, we need a consistent, reproducible development environment that:
- Runs PostgreSQL 18 with PostGIS 3.5 extension
- Supports both Go and Node.js development
- Can be easily started and stopped via Docker Compose
- Includes proper directory structure for mono-repo organization
- Has development tooling configured (linting, formatting, etc.)

## User Stories

- **US-DEV-1**: As a developer, I want to start all services with a single command so that I can begin development quickly
- **US-DEV-2**: As a developer, I want PostgreSQL with PostGIS running in Docker so that I have spatial database capabilities
- **US-DEV-3**: As a developer, I want a clear project structure so that I know where to place backend and frontend code
- **US-DEV-4**: As a developer, I want development tooling configured so that code quality is maintained automatically

## Technical Approach

### Directory Structure
```
atlas/
├── docker-compose.yml
├── .env.example
├── .gitignore
├── README.md
├── api/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── config/
│   │   ├── models/
│   │   ├── handlers/
│   │   ├── services/
│   │   └── database/
│   ├── go.mod
│   ├── go.sum
│   └── .golangci.yml
├── web/
│   ├── nextjs/
│   │   ├── package.json
│   │   ├── tsconfig.json
│   │   ├── next.config.js
│   │   └── app/
│   └── vue/
│       ├── package.json
│       ├── tsconfig.json
│       ├── vite.config.ts
│       └── src/
├── scripts/
│   ├── import-parcels.sh
│   └── setup-db.sh
└── docs/
    ├── delivery/
    └── technical/
```

### Docker Services
- **postgres-postgis**: PostgreSQL 18.0 + PostGIS 3.5
  - Port: 5432
  - Volume for data persistence
  - Health checks configured
- **api** (future): Go API service
- **nextjs** (future): Next.js frontend
- **vue** (future): Vue.js frontend

### Technology Versions
- PostgreSQL: 18.0
- PostGIS: 3.5
- Go: 1.25+
- Node.js: 20+ (LTS)
- Docker: 25.0+
- Docker Compose: 2.30+

## UX/UI Considerations

N/A - Infrastructure setup only

## Acceptance Criteria

1. ✅ Docker Compose file exists and defines all required services
2. ✅ PostgreSQL + PostGIS container starts successfully
3. ✅ Can connect to PostgreSQL on localhost:5432
4. ✅ PostGIS extension is available and queryable
5. ✅ Directory structure created for backend, frontends, and docs
6. ✅ .env.example file with all required environment variables
7. ✅ .gitignore properly excludes node_modules, vendor, .env, etc.
8. ✅ README.md with setup instructions and project overview
9. ✅ Go module initialized in backend/ directory
10. ✅ package.json initialized in both frontend directories
11. ✅ golangci-lint configuration present
12. ✅ ESLint configuration present for frontends

## Dependencies

- Docker 25.0+ and Docker Compose 2.30+ installed on development machine
- Go 1.25+ installed
- Node.js 20+ (LTS) installed
- pnpm package manager [[memory:2879566]]

## Open Questions

1. Should we include pgAdmin or another database UI tool in Docker Compose?
2. Do we need a reverse proxy (nginx/Caddy) for local development?
3. Should we include sample .env values or keep them empty?

## Related Tasks

See [tasks.md](./tasks.md) for the detailed task breakdown.

---

**Status**: Agreed  
**Created**: 2025-10-19  
**Last Updated**: 2025-10-19

