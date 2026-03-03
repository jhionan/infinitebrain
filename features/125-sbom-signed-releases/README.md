# T-125 — SBOM + Signed Releases

## Overview

Every release ships with a signed Software Bill of Materials (SBOM). Container images are
signed with cosign (sigstore). Dependency provenance is verifiable. Supply chain attacks are
detectable.

## CI/CD Additions

```yaml
# .github/workflows/release.yml

- name: Build container image
  uses: docker/build-push-action@v5
  with:
    push: true
    tags: ghcr.io/infinitebrain/server:${{ github.ref_name }}

- name: Generate SBOM (Syft)
  uses: anchore/sbom-action@v0
  with:
    image: ghcr.io/infinitebrain/server:${{ github.ref_name }}
    format: spdx-json
    output-file: sbom.spdx.json
    upload-artifact: true

- name: Sign image (cosign / sigstore)
  run: |
    cosign sign --yes ghcr.io/infinitebrain/server:${{ github.ref_name }}

- name: Attest SBOM
  run: |
    cosign attest --yes \
      --predicate sbom.spdx.json \
      --type spdxjson \
      ghcr.io/infinitebrain/server:${{ github.ref_name }}
```

## go.sum Pinning Policy

All Go dependencies are pinned to exact versions in `go.sum`. Dependabot opens PRs for
security updates weekly. No `go get -u ./...` in CI — only explicit version bumps.

## Docker Image Pinning

```yaml
# docker-compose.yml — all images pinned to SHA, not just tag
image: pgvector/pgvector@sha256:<digest>
image: valkey/valkey@sha256:<digest>
```

## Verification (for users)

```bash
# Verify the image signature before pulling
cosign verify ghcr.io/infinitebrain/server:v1.0.0 \
  --certificate-identity-regexp="https://github.com/infinitebrain/infinite_brain" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"

# Verify the SBOM attestation
cosign verify-attestation --type spdxjson \
  ghcr.io/infinitebrain/server:v1.0.0
```

## Acceptance Criteria

- [ ] `release.yml` workflow builds, signs, and attests every tagged release
- [ ] SBOM artifact uploaded to GitHub release as `sbom.spdx.json`
- [ ] cosign signature verifiable with keyless signing (OIDC-based, no key to manage)
- [ ] Dependabot configured for Go modules + Docker images
- [ ] Docker Compose pins all images to SHA digests
- [ ] `make verify-release TAG=v1.0.0` target runs cosign verification locally
- [ ] SECURITY.md documents the verification steps

## Dependencies

- T-108 (Dockerfile — image must exist to sign)
- T-091 (CI pipeline — release workflow addition)
