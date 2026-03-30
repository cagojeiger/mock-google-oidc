# mock-google-oidc

A tiny Google-compatible OIDC mock for local development and integration tests.

Use it when you want to test Google login flows locally without real Google accounts, cloud console setup, or external network dependencies.

Korean README: [README.ko.md](README.ko.md)

## What It Looks Like

![demo](docs/demo.gif)

```text
[localhost:4180]
    -> [mock-google-oidc login page]
    -> [Login]
    -> [oauth2-proxy callback]
    -> [nginx]
```

## Quick Start

```bash
docker compose up --build
```

Open:

```text
http://localhost:4180
```

Then:

```text
1. oauth2-proxy redirects to mock-google-oidc
2. enter email and name
3. click Login
4. return to oauth2-proxy
5. reach nginx upstream
```

Services:

| Service | Port | Purpose |
| --- | --- | --- |
| mock-google-oidc | 9082 | Mock OIDC provider |
| oauth2-proxy | 4180 | Example relying party |
| upstream (nginx) | 9080 | Protected upstream |

## When To Use This

- You use Google OAuth or OIDC in your app
- You want local login testing without real Google accounts
- You want to test Authorization Code + PKCE flows
- You want a simple mock provider for `oauth2-proxy` or similar clients

## What It Supports

- Authorization Code flow
- PKCE (`S256`, `plain`)
- `id_token` signing with RS256
- JWKS endpoint
- OpenID Connect discovery
- `userinfo` endpoint
- `nonce`
- deterministic `sub` from email
- single-use authorization codes
- error simulation from the login page

## What It Does Not Do

- refresh tokens
- real Google authentication
- HTTPS production setup
- strict `client_secret` value validation
- persistent storage

## Main Endpoints

| Endpoint | Purpose |
| --- | --- |
| `GET /o/oauth2/v2/auth` | Authorization page |
| `POST /o/oauth2/v2/auth` | Submit login form |
| `POST /token` | Exchange code for tokens |
| `GET /v1/userinfo` | User profile |
| `GET /.well-known/openid-configuration` | Discovery |
| `GET /oauth2/v3/certs` | JWKS |
| `GET /health` | Health check |

## Example With oauth2-proxy

This repository already includes a ready-to-run `oauth2-proxy` example in [`docker-compose.yml`](/Users/kangheeyong/project/test-idp/docker-compose.yml).

Flow:

```text
localhost:4180
  -> oauth2-proxy
  -> mock-google-oidc login page
  -> Login
  -> oauth2-proxy callback
  -> nginx
```

Important endpoints in that setup:

```text
issuer:       http://mock-google-oidc:9082
login url:    http://localhost:9082/o/oauth2/v2/auth
token url:    http://mock-google-oidc:9082/token
jwks url:     http://mock-google-oidc:9082/oauth2/v3/certs
userinfo url: http://mock-google-oidc:9082/v1/userinfo
```

## Run Only The Mock Provider

```bash
docker run -p 9082:9082 \
  -e LISTEN_ADDR=:9082 \
  -e PUBLIC_URL=http://localhost:9082 \
  ghcr.io/cagojeiger/mock-google-oidc:latest
```

Then configure your app to use:

```text
issuer:       http://localhost:9082
auth url:     http://localhost:9082/o/oauth2/v2/auth
token url:    http://localhost:9082/token
userinfo url: http://localhost:9082/v1/userinfo
jwks url:     http://localhost:9082/oauth2/v3/certs
```

## Error Testing

From the login page, open `Advanced (Response Mode)` and choose:

- `Normal`
- `Deny`
- `Token Error`
- `Userinfo Error`

This is useful when you want to verify how your app handles login failures.

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `LISTEN_ADDR` | `:8082` | Server bind address |
| `PUBLIC_URL` | `http://localhost:8082` | Issuer URL used in discovery and tokens |

## Development

Run tests:

```bash
go test ./...
```

This repository uses only the Go standard library.

## Project Layout

```text
.
├── main.go
├── handler.go
├── store.go
├── jwt.go
├── template.go
├── handler_test.go
├── flow_test.go
├── Dockerfile
├── docker-compose.yml
└── docs/
```
