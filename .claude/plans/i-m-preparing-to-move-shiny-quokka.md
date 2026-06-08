# Plan: Document manual image-bundle generation for IL6

## Context

Herald container images normally reach IL6 via the [Herald CDS Proxy](https://github.com/s2va/herald)'s `image-release.yaml` workflow: it `crane pull`s the GHCR release image into `image.tar`, bundles it as `herald-v<tag>.tar.gz`, and ships it across the cross-domain solution. The IL6 side of that flow is already documented in `deploy/il6.md` under **CDS Artifacts → Image Bundle** (extract → `docker load` → tag → push).

The user is preparing to push resources up to IL6 and wants to bypass the CDS workflow, generating the `herald-v<tag>.tar.gz` bundle manually on a connected workstation so it can be carried up by hand. Today the *consumer* (import) side is documented but the *producer* (bundle generation) side lives only in the proxy repo's workflow YAML. This plan adds the missing producer-side instructions so the manual path is fully self-documented in Herald's deploy guide.

Outcome: an operator with internet access can reproduce the exact bundle the CDS workflow would have produced, without the proxy repo or the CDS pipeline.

## File to modify

- `deploy/il6.md` — single file change.

## Change

In `deploy/il6.md`, within the **CDS Artifacts → Image Bundle (`herald-v<tag>.tar.gz`)** subsection (lines ~152–164), add a new `####`-level subsection **after** the existing extract/tag/push block (after line 164), titled something like **"Generating the bundle manually (in lieu of CDS)"**.

The new content must produce a bundle byte-compatible with the existing import steps — i.e. a gzip tarball whose root contains `image.tar`, loadable via `docker load -i image.tar`. This mirrors the proxy workflow's `crane pull … staging/image.tar` followed by `tar czf … .` from inside `staging/`.

Document both tools (per user choice), leading with `crane` to match the workflow exactly.

First, a short **Prerequisite: install crane** note. `crane` is part of [`google/go-containerregistry`](https://github.com/google/go-containerregistry) — an open-source, Google-maintained project that is the de facto registry-interaction toolkit in the cloud-native ecosystem (used by `ko`, Tekton, and many CI pipelines). It talks to registries over the OCI distribution API directly, so it needs **no Docker daemon** — which is why the CDS proxy workflow uses it. Operators who already have Docker and prefer not to add a tool can skip straight to Option B; the resulting bundle is equivalent.

Provide cross-platform install options:

```bash
# Linux / macOS (download the release binary; matches the CDS workflow)
OS=$(uname -s); ARCH=$(uname -m)   # e.g. Linux x86_64, Darwin arm64
curl -sL "https://github.com/google/go-containerregistry/releases/latest/download/go-containerregistry_${OS}_${ARCH}.tar.gz" \
  | tar xz -C "$HOME/.local/bin" crane
crane version

# Any platform with a Go toolchain
go install github.com/google/go-containerregistry/cmd/crane@latest

# Homebrew (macOS / Linuxbrew)
brew install crane
```

> **Note:** the release archive name encodes OS/arch (`go-containerregistry_Linux_x86_64.tar.gz`, `go-containerregistry_Darwin_arm64.tar.gz`, etc.). Ensure `$HOME/.local/bin` (or the chosen target) is on `PATH`.

Then the bundle generation:

```bash
# Option A — crane (matches the CDS workflow; no Docker daemon required)
mkdir -p staging
crane pull ghcr.io/jaimestill/herald:<tag> staging/image.tar
tar czf herald-v<tag>.tar.gz -C staging .
sha256sum herald-v<tag>.tar.gz > herald-v<tag>.sha256
```

```bash
# Option B — docker (when a Docker daemon is available)
mkdir -p staging
docker pull ghcr.io/jaimestill/herald:<tag>
docker save ghcr.io/jaimestill/herald:<tag> -o staging/image.tar
tar czf herald-v<tag>.tar.gz -C staging .
sha256sum herald-v<tag>.tar.gz > herald-v<tag>.sha256
```

Then add a short note that:
- `<tag>` is the version without the `v` prefix (e.g. `0.6.0`), matching the existing convention — the bundle filename keeps the `v` (`herald-v0.6.0.tar.gz`), the image tag does not (`ghcr.io/jaimestill/herald:0.6.0`). This mirrors the workflow's `IMAGE_TAG=${VERSION#v}` behavior.
- The resulting `herald-v<tag>.tar.gz` (and `.sha256`) is the same artifact the CDS workflow uploads; carry it up and import with the extract/tag/push steps already shown above.
- A `> **Note:**` callout that this manual path skips CDS scanning/transfer governance and should only be used when the CDS workflow is unavailable.

## Conventions to match (from existing `deploy/il6.md`)

- `##` top-level / `###` step sections / `####` for the new nested subsection under "Image Bundle".
- Fenced code blocks tagged `bash` (the producer runs on a connected Linux/macOS workstation; the existing import block uses `powershell` for the IL6 side — keep that distinction).
- Angle-bracket placeholders: `<tag>`, `<acr-name>`, `<il6-domain-root>`.
- Backticked inline identifiers; `> **Note:**` / `> **Important:**` blockquote callouts.
- No emoji; direct imperative tone.

## Verification

- Re-read the edited `deploy/il6.md` section to confirm heading nesting, language tags, and placeholder style match surrounding content.
- Sanity-check the bundle layout claim: `tar czf herald-v<tag>.tar.gz -C staging .` produces an archive whose root is `./image.tar`, so the existing `tar xzf … && docker load -i image.tar` import steps consume it unchanged.
- (Optional, off-network) Dry-run the crane/docker producer commands against a current GHCR tag (e.g. `0.6.0`) on a connected machine, then `tar tzf herald-v0.6.0.tar.gz` to confirm `./image.tar` is present and `docker load -i image.tar` succeeds — validates parity with the CDS-produced bundle.
- Confirm `sha256sum -c herald-v<tag>.sha256` validates the bundle (matches the workflow's checksum step).

## Notes

- Docs-only change to a deploy guide; per project conventions this is a deploy/docs edit with no container image impact — no CHANGELOG/dev-tag needed.
- No change to the proxy repo (`~/code/_s2va/herald`); it remains the source of truth for the automated path.
