# Changelog

## v0.1.0-dev.3.21

- Add read-only blob storage query endpoints (list, find, download) with marker-based pagination, configurable max list size, and wildcard path parameter routing (#21)

## v0.1.0-dev.3.17

- Add document HTTP handlers (List, Find, Search, Upload, Delete), multipart upload with pdfcpu page count extraction, configurable max upload size, and API route wiring (#17)
- Replace OpenAPI/Scalar with API Cartographer — markdown-based API documentation with executable curl examples (#17)

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
