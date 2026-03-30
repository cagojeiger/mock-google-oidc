# Spec 001: mock-google-oidc 개요

## 목적

Google OAuth 2.0 / OpenID Connect **호환 mock Identity Provider**.
패스워드 없이 이메일만 입력하면 로그인되며, Google과 호환되는 형식의 응답을 반환한다.

## 한 줄 요약

```
Google OIDC를 흉내내되, 로그인 화면에서 이메일만 입력하면 바로 인증 완료.
```

## 성격

```
이것은 "Google과 동일한 OIDC Provider"가 아니라
"Google OIDC 호환 지향 mock"이다.

- HTTPS가 아닌 HTTP로 동작한다 (로컬 개발용)
- client_id/client_secret 검증을 하지 않는다
- refresh_token을 지원하지 않는다
- 구현 범위는 Authorization Code + PKCE 플로우에 한정한다
- authorization code는 1회용 (OAuth 스펙 준수)
```

## 설계 원칙

```
1. Google OIDC Authorization Code 플로우와 호환되는 엔드포인트/요청/응답 형식
2. PKCE (S256, plain) 지원
3. 완전 독립 실행 — 외부 서비스, DB 없음
4. 패스워드 없는 로그인 — 이메일 + 이름 입력 → 즉시 인증
```

## Google과의 대응 관계

| Google | mock-google-oidc | 비고 |
|--------|-----------------|------|
| `accounts.google.com/o/oauth2/v2/auth` | `/o/oauth2/v2/auth` | 동일 경로 |
| `oauth2.googleapis.com/token` | `/token` | POST body 형식 호환 |
| `openidconnect.googleapis.com/v1/userinfo` | `/v1/userinfo` | 응답 JSON 형식 호환 |
| `accounts.google.com/.well-known/openid-configuration` | `/.well-known/openid-configuration` | Discovery 문서 |
| `www.googleapis.com/oauth2/v3/certs` | `/oauth2/v3/certs` | JWKS (공개키) |
| Google 로그인 화면 (이메일 + 패스워드) | 이메일 + 이름 입력 화면 (패스워드 없음) | 유일한 UX 차이 |

## 호환 범위

| 기능 | 지원 | 비고 |
|------|------|------|
| Authorization Code Grant | 예 | |
| PKCE (S256, plain) | 예 | |
| id_token (RS256 JWT) | 예 | JWKS로 검증 가능 |
| userinfo endpoint | 예 | |
| Discovery document | 예 | |
| nonce | 예 | id_token에 포함 |
| email_verified | 예 | 항상 true |
| authorization code 1회용 | 예 | ConsumeCode로 구현 |
| grant_type 검증 | 예 | authorization_code 외 거부 |
| refresh_token | **아니오** | discovery에서 미광고 |
| client_id/secret 검증 | **아니오** | 모든 값 수용 |
| HTTPS | **아니오** | 로컬 HTTP만 |

## oauth2-proxy 호환 체크리스트

```
oauth2-proxy가 기대하는 것     mock-google-oidc 지원
─────────────────────────────────────────────────
discovery 읽기                 ✓ /.well-known/openid-configuration
auth endpoint redirect         ✓ /o/oauth2/v2/auth
token endpoint code redeem     ✓ /token (+ PKCE + code 1회용)
id_token issuer 검증           ✓ issuer = PUBLIC_URL
id_token aud 검증              ✓ aud = client_id
id_token RS256 서명 검증       ✓ JWKS /oauth2/v3/certs
email claim                    ✓ 항상 포함
email_verified                 ✓ 항상 true
userinfo endpoint              ✓ /v1/userinfo
nonce                          ✓ id_token에 포함
PKCE code_challenge            ✓ S256, plain
```

## 기술 스택

- **언어**: Go
- **UI**: html/template (서버 사이드 렌더링, 외부 의존성 없음)
- **JWT 서명**: RS256 (서버 시작 시 키 생성)
- **상태**: 인메모리 (DB 없음, 재시작 시 초기화)
- **외부 의존성**: 없음 (표준 라이브러리만)
