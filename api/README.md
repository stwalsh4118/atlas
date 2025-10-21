# Atlas API

Go backend for the Atlas Property Boundary Viewer application.

## Prerequisites

- Go 1.25+
- Docker & Docker Compose
- golang-migrate CLI (installed via `go install`)
- PostgreSQL 18 with PostGIS 3.5+ (via Docker Compose)

## Project Structure

```
api/
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── database/        # Database connection
│   ├── handlers/        # HTTP request handlers
│   ├── middleware/      # HTTP middleware
│   ├── models/          # Data models (GORM)
│   ├── repository/      # Database operations
│   └── services/        # Business logic
├── migrations/          # Database migration files
├── go.mod
├── go.sum
├── Makefile            # Development commands
└── README.md
```

## Getting Started

### 1. Start PostgreSQL with Docker

```bash
# From project root
docker-compose up -d
```

### 2. Install golang-migrate CLI

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Make sure `$GOPATH/bin` or `~/go/bin` is in your PATH.

### 3. Install Dependencies

```bash
cd api
go mod tidy
```

## Database Migrations

This project uses [golang-migrate](https://github.com/golang-migrate/migrate) for database schema management.

### Quick Reference

```bash
# Create a new migration
make migrate-create NAME=add_users_table

# Run all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Check current migration version
make migrate-version

# View all available commands
make help
```

### Detailed Migration Commands

#### Create a New Migration

```bash
make migrate-create NAME=your_migration_name
```

This creates two files:
- `migrations/NNNNNN_your_migration_name.up.sql` - Applied when migrating forward
- `migrations/NNNNNN_your_migration_name.down.sql` - Applied when rolling back

#### Apply Migrations

```bash
# Apply all pending migrations
make migrate-up

# Migrate to specific version
make migrate-goto VERSION=3
```

#### Rollback Migrations

```bash
# Rollback the last migration
make migrate-down

# Rollback all migrations (WARNING: destructive)
make migrate-down-all

# Drop everything (WARNING: extremely destructive)
make migrate-drop
```

#### Check Migration Status

```bash
# Show current migration version
make migrate-version
```

#### Force Migration Version

If migrations get out of sync (e.g., after manual database changes):

```bash
make migrate-force VERSION=3
```

###  Database Connection Configuration

Override default connection settings with environment variables:

```bash
# Example with custom settings
DB_HOST=host.docker.internal \
DB_PORT=5432 \
DB_USER=postgres \
DB_PASSWORD=your_password \
DB_NAME=atlas \
make migrate-up
```

If you run inside WSL2 with Docker Desktop, Postgres is exposed through the Windows host. Use `host.docker.internal` as the default hostname. If you run Docker natively on Linux, override `DB_HOST=localhost` in your environment.

### Migration Best Practices

1. **Always create both up and down migrations**
   - Up migration: Schema changes
   - Down migration: How to reverse those changes

2. **Test migrations before committing**
   ```bash
   make migrate-up    # Apply
   make migrate-down  # Rollback
   make migrate-up    # Re-apply
   ```

3. **Keep migrations small and focused**
   - One logical change per migration
   - Easier to review and rollback

4. **Never modify existing migrations**
   - Once merged to main, migrations are immutable
   - Create a new migration to fix issues

5. **Use transactions where possible**
   ```sql
   BEGIN;
   -- Your changes here
   COMMIT;
   ```

6. **Include comments in migration files**
   ```sql
   -- Add spatial index for parcel geometry lookups
   -- This index improves ST_Contains query performance
   CREATE INDEX idx_parcels_geom ON tax_parcels USING GIST(geom);
   ```

### Troubleshooting

#### Connection Refused

If you see "connection refused" errors:

```bash
# Check if PostgreSQL is running
docker-compose ps

# Start if not running
docker-compose up -d

# Check logs
docker-compose logs postgres
```

#### Authentication Failed

If local psql connections fail but docker exec works, run migrations via docker exec:

```bash
# Alternative method using docker exec
docker exec atlas-postgres psql -U postgres -d atlas -f migrations/000001_your_migration.up.sql
```

#### Dirty Migration State

If migrations are in a "dirty" state (partially applied):

1. Fix the issue manually in the database
2. Force the version:
   ```bash
   make migrate-force VERSION=N
   ```

## Development

### Run the API Server

**With Hot Reload (Recommended for Development):**
```bash
make dev
```

This uses [Air](https://github.com/cosmtrek/air) for automatic reloading when files change. Air is configured via `.air.toml` to watch `.go` files and rebuild automatically.

**Without Hot Reload:**
```bash
make run
```

### Build

```bash
make build
```

The binary will be created at `bin/server`.

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Tidy dependencies
make tidy
```

## Environment Variables

Create a `.env` file in the api directory:

```bash
# Server Configuration
PORT=8080
ENV=development

# Database Configuration  
DB_HOST=host.docker.internal
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=atlas
DB_POOL_MIN=2
DB_POOL_MAX=10

# CORS Configuration
CORS_ORIGINS=http://localhost:3000,http://localhost:3001
```

## GORM + PostGIS Integration

This project uses GORM with custom PostGIS types. See the integration guide:

- [GORM + PostGIS Integration Guide](../docs/delivery/2/2-1-gorm-postgis-guide.md)

Key points:
- Custom `Polygon` type with Scanner/Valuer interfaces
- GORM for standard CRUD operations
- Raw SQL for spatial queries (ST_Contains, ST_Intersects, etc.)
- GeoJSON parsing for Montgomery County data

## API Documentation

Once implemented, API documentation will be available at:
- Swagger UI: `http://localhost:8080/swagger`
- OpenAPI Spec: `http://localhost:8080/swagger/doc.json`

## Contributing

1. Create a branch for your changes
2. Write tests for new functionality
3. Ensure all tests pass: `make test`
4. Run linter: `make lint`
5. Create a pull request

## License

MIT License - See LICENSE file for details

