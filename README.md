# mock-google-oidc

A mock OIDC provider for testing Google login flows locally.

No real Google accounts, no cloud console setup, no external network required.

Korean: [README.ko.md](README.ko.md)

![demo](docs/demo.gif)

## Quick Start

```bash
docker compose up --build
```

Open `http://localhost:4180` → mock login → nginx upstream.

## Endpoints

| Path | Purpose |
| --- | --- |
| `/.well-known/openid-configuration` | OIDC Discovery |
| `/o/oauth2/v2/auth` | Authorization (Google-style path) |
| `/token` | Token exchange |
| `/v1/userinfo` | User profile |
| `/oauth2/v3/certs` | JWKS |

## Features

- Authorization Code + PKCE (`S256`, `plain`)
- RS256 `id_token` + JWKS
- `coreos/go-oidc` conformance tests passing
- Google-compatible paths
- Error simulation (Deny, Token Error, Userinfo Error)

## Test

```bash
go test ./...
```

81 tests: pure functions → handlers → integration flows → OIDC conformance (C1–C7).

## Project Layout

```
cmd/mock-google-oidc/main.go     # Entrypoint
internal/oidc/
  handler.go                     # HTTP handlers
  store.go                       # In-memory store
  jwt.go                         # RSA + JWT
  validate.go                    # Pure validation functions
  conformance_test.go            # coreos/go-oidc conformance
docs/                            # Spec documents
```

## Docs

Start from [docs/001-overview.md](docs/001-overview.md).

```
001-overview.md              Project goals
002-reference-specs.md       Reference specs
003-endpoints.md             HTTP contract
004-google-compatibility.md  Google compatibility scope
005-flow.md                  Auth flow
006-conformance-boundary.md  MUST / SHOULD / non-compliant
007-development.md           Running and developing
```
