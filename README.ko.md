# mock-google-oidc

로컬 개발과 통합 테스트를 위한 아주 작은 Google 호환 OIDC mock입니다.

실제 Google 계정 없이, 클라우드 콘솔 설정 없이, 외부 네트워크 의존 없이 Google 로그인 플로우를 테스트하고 싶을 때 사용합니다.

영문 README: [README.md](README.md)

## 한눈에 보기

![demo](docs/demo.gif)

```text
[브라우저]
    |
    v
[내 앱 / oauth2-proxy]
    |
    v
[mock-google-oidc]
    |
    +-> 이메일 + 이름 입력
    +-> Login 클릭
    +-> code / token / userinfo 반환
```

```text
[localhost:4180]
    -> [mock-google-oidc 로그인 화면]
    -> [Login]
    -> [oauth2-proxy callback]
    -> [nginx]
```

## 빠른 시작

```bash
docker compose up --build
```

브라우저에서:

```text
http://localhost:4180
```

흐름:

```text
1. oauth2-proxy가 mock-google-oidc로 리다이렉트
2. 이메일과 이름 입력
3. Login 클릭
4. oauth2-proxy로 돌아감
5. nginx upstream 도달
```

서비스:

| 서비스 | 포트 | 용도 |
| --- | --- | --- |
| mock-google-oidc | 9082 | Mock OIDC Provider |
| oauth2-proxy | 4180 | 예제 RP |
| upstream (nginx) | 9080 | 보호된 업스트림 |

## 이럴 때 쓰면 됨

- 앱에서 Google OAuth 또는 OIDC를 사용 중일 때
- 실제 Google 계정 없이 로컬 로그인 테스트를 하고 싶을 때
- Authorization Code + PKCE 플로우를 검증하고 싶을 때
- `oauth2-proxy` 같은 클라이언트와 간단히 붙여보고 싶을 때

## 지원하는 것

- Authorization Code 플로우
- PKCE (`S256`, `plain`)
- RS256 `id_token`
- JWKS endpoint
- OpenID Connect discovery
- `userinfo` endpoint
- `nonce`
- 이메일 기반 deterministic `sub`
- 1회용 authorization code
- 로그인 화면에서 에러 시나리오 선택

## 지원하지 않는 것

- refresh token
- 실제 Google 인증
- 프로덕션용 HTTPS 구성
- 엄격한 `client_secret` 값 검증
- 영속 저장소

## 주요 엔드포인트

| 엔드포인트 | 용도 |
| --- | --- |
| `GET /o/oauth2/v2/auth` | 로그인 화면 |
| `POST /o/oauth2/v2/auth` | 로그인 폼 제출 |
| `POST /token` | code를 token으로 교환 |
| `GET /v1/userinfo` | 사용자 정보 |
| `GET /.well-known/openid-configuration` | Discovery |
| `GET /oauth2/v3/certs` | JWKS |
| `GET /health` | Health check |

## oauth2-proxy 예제

이 저장소에는 바로 실행 가능한 `oauth2-proxy` 예제가 [`docker-compose.yml`](/Users/kangheeyong/project/test-idp/docker-compose.yml)에 포함되어 있습니다.

흐름:

```text
localhost:4180
  -> oauth2-proxy
  -> mock-google-oidc 로그인 화면
  -> Login
  -> oauth2-proxy callback
  -> nginx
```

## mock provider만 따로 실행

```bash
docker run -p 9082:9082 \
  -e LISTEN_ADDR=:9082 \
  -e PUBLIC_URL=http://localhost:9082 \
  ghcr.io/cagojeiger/mock-google-oidc:latest
```

앱 설정:

```text
issuer:       http://localhost:9082
auth url:     http://localhost:9082/o/oauth2/v2/auth
token url:    http://localhost:9082/token
userinfo url: http://localhost:9082/v1/userinfo
jwks url:     http://localhost:9082/oauth2/v3/certs
```

## 에러 테스트

로그인 화면에서 `Advanced (Response Mode)`를 열면:

- `Normal`
- `Deny`
- `Token Error`
- `Userinfo Error`

를 선택할 수 있습니다.

## 설정

| 변수 | 기본값 | 설명 |
| --- | --- | --- |
| `LISTEN_ADDR` | `:8082` | 서버 바인딩 주소 |
| `PUBLIC_URL` | `http://localhost:8082` | discovery와 token에 반영되는 issuer URL |

## 개발

```bash
go test ./...
```

이 저장소는 Go 표준 라이브러리만 사용합니다.
