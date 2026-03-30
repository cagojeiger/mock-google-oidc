# mock-google-oidc

Google OIDC 호환 mock Identity Provider. 패스워드 없이 이메일만 입력하면 로그인된다.

## 왜 필요한가

- Google OAuth를 사용하는 서비스를 로컬에서 개발/테스트할 때
- 실제 Google 계정 없이 OIDC 인증 플로우를 테스트할 때
- oauth2-proxy, authgate 같은 OIDC RP와 연동 테스트할 때

## 빠른 시작

```bash
docker compose up --build
```

3개 서비스가 시작된다:

| 서비스 | 포트 | 설명 |
|--------|------|------|
| mock-google-oidc | 9082 | Mock OIDC Provider |
| oauth2-proxy | 4180 | OIDC RP (통합 테스트용) |
| upstream (nginx) | 9080 | 인증 후 도달하는 백엔드 |

브라우저에서 `http://localhost:4180` 접속하면 전체 플로우를 체험할 수 있다:

```
localhost:4180 → oauth2-proxy → mock-google-oidc 로그인 → Login 클릭 → nginx
```

## mock-google-oidc 단독 사용

다른 OIDC RP와 연동하려면 mock-google-oidc만 사용하면 된다.

### 엔드포인트

| 엔드포인트 | Google 대응 |
|-----------|------------|
| `GET /o/oauth2/v2/auth` | Authorization (로그인 화면) |
| `POST /token` | Token 교환 |
| `GET /v1/userinfo` | 사용자 정보 |
| `GET /.well-known/openid-configuration` | OIDC Discovery |
| `GET /oauth2/v3/certs` | JWKS (공개키) |
| `GET /health` | 헬스 체크 |

### RP 설정 예시

기존 Google OAuth 설정을 이렇게 바꾸면 된다:

```
# Google → mock-google-oidc
OAUTH_ISSUER_URL=http://localhost:9082
OAUTH_AUTH_URL=http://localhost:9082/o/oauth2/v2/auth
OAUTH_TOKEN_URL=http://localhost:9082/token
OAUTH_USERINFO_URL=http://localhost:9082/v1/userinfo
OAUTH_JWKS_URL=http://localhost:9082/oauth2/v3/certs
OAUTH_CLIENT_ID=anything      # 검증 안 함
OAUTH_CLIENT_SECRET=anything  # 검증 안 함
```

### 환경 변수

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `LISTEN_ADDR` | `:8082` | 서버 바인딩 주소 |
| `PUBLIC_URL` | `http://localhost:8082` | issuer URL (discovery에 반영) |

docker-compose에서는 `9082`로 오버라이드된다.

## 로그인 화면

브라우저에서 직접 확인:

```
http://localhost:9082/o/oauth2/v2/auth?redirect_uri=http://localhost:9082/health&state=test
```

- Email과 Name이 미리 채워져 있어서 **Login만 클릭**하면 된다
- 같은 이메일 → 같은 sub (기존 유저로 인식)
- 다른 이메일 → 다른 sub (새 유저로 인식)

### Response Mode (에러 테스트)

로그인 화면 하단 Advanced에서 선택:

| 모드 | 동작 |
|------|------|
| Normal | 정상 로그인 (기본) |
| Deny | access_denied 반환 |
| Token Error | /token에서 500 |
| Userinfo Error | /userinfo에서 500 |

## 지원 기능

| 기능 | 지원 |
|------|------|
| Authorization Code Grant | O |
| PKCE (S256, plain) | O |
| id_token (RS256 JWT) | O |
| JWKS | O |
| userinfo | O |
| Discovery | O |
| nonce | O |
| email_verified | O (항상 true) |
| code 1회용 | O |
| grant_type 검증 | O |
| client_id 매칭 | O |
| client_secret_post | O |
| client_secret_basic | O |
| PKCE 실패 시 code 보존 | O |
| refresh_token | X |
| client_secret 값 검증 | X (존재만 확인) |
| HTTPS | X (HTTP만) |

## 테스트

```bash
# 단위 + 플로우 테스트 (34개)
go test ./...

# Docker로 전체 스택 테스트
docker compose up --build
# 브라우저에서 http://localhost:4180 접속
```

## 상세 스펙

- [001-overview.md](docs/001-overview.md) — 목적, 호환 범위
- [002-endpoints.md](docs/002-endpoints.md) — 엔드포인트 상세
- [003-ui.md](docs/003-ui.md) — 로그인 화면
- [004-flow.md](docs/004-flow.md) — 전체 흐름, Google 비교
- [005-development.md](docs/005-development.md) — 개발, 테스트, 프로젝트 구조
