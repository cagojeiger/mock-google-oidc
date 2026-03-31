# Spec 007: 개발과 실행

## 실행 목적

개발 환경에서는 아래 2가지를 빠르게 확인할 수 있어야 한다.

```text
1. mock provider 단독 동작
2. oauth2-proxy와의 Google 로그인 대체 통합
```

## 빠른 실행

```bash
docker compose up --build
```

구성:

| 서비스 | 포트 | 설명 |
| --- | --- | --- |
| mock-google-oidc | 9082 | Google 호환 mock OIDC Provider |
| oauth2-proxy | 4180 | 예제 RP |
| upstream | 9080 | 인증 후 도달하는 대상 |

## 실행 구조

```text
[브라우저 localhost:4180]
          |
          v
[oauth2-proxy]
          |
          v
[mock-google-oidc :9082]
          |
          v
[upstream nginx]
```

## 환경 변수

| 변수 | 기본값 | docker-compose 값 | 설명 |
| --- | --- | --- | --- |
| `LISTEN_ADDR` | `:8082` | `:9082` | 서버 바인딩 주소 |
| `PUBLIC_URL` | `http://localhost:8082` | `http://localhost:9082` | issuer와 discovery에 쓰는 외부 URL |

## mock provider만 단독 확인

브라우저에서 아래 URL을 열면 로그인 화면이 보인다.

```text
http://localhost:9082/o/oauth2/v2/auth?redirect_uri=http://localhost:9082/health&state=test123&scope=openid&response_type=code
```

## 권장 개발 원칙

```text
- 문서에 먼저 계약을 적는다
- 핸들러 테스트와 플로우 테스트를 같이 추가한다
- 커밋 전 go test ./... 통과를 목표로 한다
- Google 호환 경로를 깨는 변경은 compatibility 문서를 먼저 갱신한다
```

## 현재 코드 구조

```text
.
├── cmd/mock-google-oidc/main.go
├── internal/oidc/handler.go
├── internal/oidc/store.go
├── internal/oidc/jwt.go
├── internal/oidc/template.go
├── internal/oidc/validate.go
├── internal/oidc/handler_test.go
├── internal/oidc/flow_test.go
├── internal/oidc/conformance_test.go
├── internal/oidc/validate_test.go
├── Dockerfile
├── docker-compose.yml
└── docs/
```

컴포넌트 관계:

```text
[main.go]
    |
    v
[RegisterHandlers]
    |
    +-> discovery
    +-> authorize
    +-> token
    +-> userinfo
    +-> jwks
    `-> health

[token/userinfo]
    |
    +-> [Store]
    `-> [KeyPair]
```

## 수동 점검 시나리오

### 1. 브라우저 흐름

```text
1. http://localhost:4180 접속
2. mock-google-oidc 로그인 화면으로 리다이렉트
3. email/name 입력 후 Login
4. callback 완료
5. upstream 도달
```

### 2. provider API 흐름

```bash
curl "http://localhost:9082/.well-known/openid-configuration"
```

```bash
curl "http://localhost:9082/oauth2/v3/certs"
```

## 버전 정책

SemVer를 사용한다.

예시:

```text
v0.x  실험 단계
v1.0  Google 로그인 테스트용 핵심 흐름 안정화
```
