# 6 - Storage Abstraction

## Summary

Added the `storage.System` interface and Azure Blob Storage implementation to `pkg/storage/`. The package provides streaming blob operations (Upload, Download, Delete, Exists) with lifecycle-coordinated container initialization. Fixed an Azurite API version compatibility issue discovered during testing.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Single file for interface + implementation | `storage.go` contains everything | Mirrors `database.go` pattern; implementation is ~150 lines |
| Streaming I/O | `io.Reader` for upload, `io.ReadCloser` for download | Documents are large PDFs; byte slices would be wasteful |
| No shutdown hook | Omitted `OnShutdown` | azblob client is stateless HTTP — no connections to drain |
| Field naming | `container` not `containerName` | Short naming per Go conventions |
| No integration tests | Unit tests only | Integration tests that depend on Azurite would silently skip in CI; unit tests cover constructor, sentinel errors, and key validation |
| Azurite `--skipApiVersionCheck` | Added to compose command | azblob SDK v1.6.4 sends API version `2026-02-06` which Azurite doesn't natively support |

## Files Modified

- `pkg/storage/storage.go` — new: System interface, azure implementation, New constructor, validateKey helper
- `pkg/storage/errors.go` — new: ErrNotFound, ErrEmptyKey, ErrInvalidKey sentinel errors
- `tests/storage/storage_test.go` — new: constructor, sentinel error, and key validation tests
- `compose/azurite.yml` — modified: added `--skipApiVersionCheck` to Azurite command
- `go.mod` / `go.sum` — modified: added `azure-sdk-for-go/sdk/storage/azblob` v1.6.4

## Patterns Established

- Storage System follows the same interface + private struct + New + Start pattern as database System
- Key validation rejects empty keys and path traversal (`..`) at the storage layer; key format enforcement is a domain concern
- Integration tests requiring external services should not be included in the standard test suite
