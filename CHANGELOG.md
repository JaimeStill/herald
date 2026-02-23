# Changelog

## v0.1.0-dev.3.16

- Add document domain core: types, mapping, repository, and system interface with blob+DB atomicity and paginated filtered queries (#16)

## v0.1.0-dev.2.13

- Add migration CLI with embedded SQL and initial documents schema (#13)

## v0.1.0-dev.1.7

- Add infrastructure assembly, API module shell, and server entry point with health/readiness probes, OpenAPI endpoint, Scalar UI, and graceful shutdown lifecycle (#7)

## v0.1.0-dev.1.6

- Add storage abstraction with Azure Blob Storage implementation, streaming blob operations, and lifecycle-coordinated container initialization (#6)

## v0.1.0-dev.1.5

- Add database toolkit: PostgreSQL connection management with lifecycle coordination, composable SQL query builder with projection mapping, generic repository helpers, domain-agnostic error mapping, and pagination types (#5)
- Fix Docker Compose network and volume configuration for local development

## v0.1.0-dev.1.4

- Establish Go module, project structure, build tooling (mise + Docker Compose), configuration system, lifecycle coordinator, OpenAPI spec infrastructure, HTTP middleware/routing/module system, and Scalar API docs (#4)
