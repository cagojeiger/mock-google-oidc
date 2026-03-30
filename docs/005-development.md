# Spec 005: 개발, 실행, 테스트

## 최초 세팅

```bash
cd mock-google-oidc
go mod init github.com/kangheeyong/mock-google-oidc
go mod tidy
```

외부 의존성: **없음**. 표준 라이브러리만 사용.

## 실행

```bash
docker compose up --build
```

### Dockerfile

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=$(cat VERSION 2>/dev/null || echo dev)" -o /mock-google-oidc .

FROM scratch
COPY --from=builder /mock-google-oidc /mock-google-oidc
ENTRYPOINT ["/mock-google-oidc"]
```

### docker-compose.yml

```yaml
services:
  mock-google-oidc:
    build: .
    ports:
      - "8082:8082"
    environment:
      LISTEN_ADDR: ":8082"
      PUBLIC_URL: "http://localhost:8082"
```

### 환경 변수

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `LISTEN_ADDR` | `:8082` | 서버 바인딩 주소 |
| `PUBLIC_URL` | `http://localhost:8082` | 외부 URL (issuer, discovery) |

## 버전

Git 태그 기반 SemVer.

```
v0.1.0  — 최초 동작 (authorize + token + userinfo + PKCE)
v1.0.0  — oauth2-proxy 통합 테스트 통과 확인
```

## 화면 단독 확인

`docker compose up --build` 후 브라우저에서 직접 열면 로그인 화면이 보인다.

```
http://localhost:8082/o/oauth2/v2/auth?redirect_uri=http://localhost:8082/health&state=test123
```

## 테스트

### 원칙

```
- 모든 코드는 테스트와 함께 작성한다.
- 커밋 전에 go test ./... 가 반드시 통과해야 한다.
- httptest 기반, 실제 포트 바인딩 없이 실행된다.
```

### 1. 엔드포인트 단위 테스트 (handler_test.go)

| 엔드포인트 | 테스트 항목 |
|-----------|-----------|
| `GET /o/oauth2/v2/auth` | 렌더링, redirect_uri/state 누락 400, login_hint |
| `POST /o/oauth2/v2/auth` | code 생성+리다이렉트, deny, email 누락 400 |
| `POST /token` | 정상 교환, invalid code, token_error, id_token 클레임 검증 |
| `POST /token` (PKCE) | S256 정상, S256 불일치, verifier 누락, plain 정상, PKCE 없이도 동작 |
| `GET /v1/userinfo` | 정상, 토큰 없음 401, 잘못된 토큰 401, userinfo_error |
| `GET /.well-known/openid-configuration` | JSON 형식, URL, refresh_token 미포함 확인 |
| `GET /oauth2/v3/certs` | JWKS 형식, RSA, RS256 |
| `GET /health` | status=ok |

### 2. 전체 플로우 테스트 (flow_test.go)

| 테스트 | 검증 |
|--------|------|
| Normal | authorize → token → id_token 클레임 → JWKS 서명 검증 → userinfo → sub 일치 |
| SameEmailSameSub | 같은 이메일 두 번 → 같은 sub |
| DifferentEmailDifferentSub | 다른 이메일 → 다른 sub |
| Deny | error=access_denied, code 없음 |
| TokenError | code 발급 → /token 500 |
| UserinfoError | token 성공 → /userinfo 500 |
| PKCE S256 | challenge/verifier 정상 → 전체 플로우 성공 |
| PKCE S256 Wrong Verifier | 잘못된 verifier → 400 |

### 3. oauth2-proxy 통합 테스트

docker-compose로 mock-google-oidc + oauth2-proxy + 간단한 upstream 앱을 함께 띄워서
실제 브라우저 인증 플로우가 동작하는지 확인한다.

```yaml
# docker-compose.integration.yml
services:
  mock-google-oidc:
    build: .
    ports:
      - "8082:8082"
    environment:
      LISTEN_ADDR: ":8082"
      PUBLIC_URL: "http://localhost:8082"

  oauth2-proxy:
    image: quay.io/oauth2-proxy/oauth2-proxy:latest
    ports:
      - "4180:4180"
    environment:
      OAUTH2_PROXY_PROVIDER: google
      OAUTH2_PROXY_CLIENT_ID: test-client
      OAUTH2_PROXY_CLIENT_SECRET: test-secret
      OAUTH2_PROXY_COOKIE_SECRET: "0123456789abcdef0123456789abcdef"
      OAUTH2_PROXY_REDIRECT_URL: "http://localhost:4180/oauth2/callback"
      OAUTH2_PROXY_OIDC_ISSUER_URL: "http://mock-google-oidc:8082"
      OAUTH2_PROXY_EMAIL_DOMAINS: "*"
      OAUTH2_PROXY_UPSTREAMS: "http://upstream:8080"
      OAUTH2_PROXY_SKIP_PROVIDER_BUTTON: "true"
      OAUTH2_PROXY_INSECURE_OIDC_ALLOW_UNVERIFIED_EMAIL: "true"
      OAUTH2_PROXY_COOKIE_SECURE: "false"
    depends_on:
      - mock-google-oidc
      - upstream

  upstream:
    image: nginx:alpine
    # 간단한 upstream 앱 (인증 확인용)
```

**검증 흐름:**
```
1. 브라우저에서 http://localhost:4180 접근
2. oauth2-proxy가 mock-google-oidc로 리다이렉트
3. 로그인 화면에서 이메일 입력 + Login
4. callback → oauth2-proxy가 id_token 검증
5. upstream 앱에 접근 성공
```

### 4. 브라우저 수동 테스트

```bash
docker compose up --build
```

```
1. 브라우저: http://localhost:8082/o/oauth2/v2/auth?redirect_uri=http://localhost:8082/health&state=test
2. Login 클릭
3. /health?code=xxx&state=test 확인
4. curl -X POST http://localhost:8082/token \
     -d "code=xxx&client_id=test&client_secret=test&redirect_uri=http://localhost:8082/health&grant_type=authorization_code"
5. curl -H "Authorization: Bearer {access_token}" http://localhost:8082/v1/userinfo
```

## 프로젝트 구조

```
mock-google-oidc/
├── main.go              # 서버 시작, 라우팅, 버전
├── handler.go           # HTTP 핸들러 (authorize, token, userinfo, discovery, certs, health)
├── store.go             # 인메모리 저장소 (codes, tokens)
├── jwt.go               # RSA 키 생성, id_token 서명, JWKS
├── template.go          # 로그인 화면 HTML 템플릿
├── handler_test.go      # 엔드포인트 단위 테스트 (PKCE 포함)
├── flow_test.go         # 전체 플로우 테스트
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── .repos/              # 참고용 리포 (gitignore)
│   └── oauth2-proxy/
├── docs/
│   ├── 001-overview.md
│   ├── 002-endpoints.md
│   ├── 003-ui.md
│   ├── 004-flow.md
│   └── 005-development.md
├── README.md
└── LICENSE
```
