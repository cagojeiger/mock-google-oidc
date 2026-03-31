# Spec 002: 기준 스펙과 준수 기준

이 프로젝트는 `Google 로그인 테스트용 mock provider`다.
하지만 프로토콜 계약은 표준 문서를 기준으로 정의하고, 경로와 사용성은 Google 호환성을 우선한다.

## 기준 문서 구조

```text
[표준 OIDC / OAuth 문서]
          |
          v
   프로토콜 계약 정의
          |
          +--> discovery
          +--> authorize
          +--> token
          +--> userinfo
          +--> jwks
          |
          v
[Google OpenID Connect 문서]
          |
          v
   Google 스타일 경로/클레임/사용 패턴 보정
          |
          v
[프로젝트 정책]
   로컬 mock 특성상 의도적 비준수 문서화
```

## 1. 1차 기준: 표준 문서

| 문서 | 용도 | 이 프로젝트에서 쓰는 범위 |
| --- | --- | --- |
| OpenID Connect Core 1.0 | OIDC 핵심 규약 | Authorization Code Flow, ID Token, UserInfo |
| OpenID Connect Discovery 1.0 | discovery metadata | provider metadata 반환 |
| RFC 6749 OAuth 2.0 | OAuth 기본 규약 | authorization code grant |
| RFC 7636 PKCE | PKCE | `plain`, `S256` |
| RFC 6750 Bearer Token Usage | bearer token | `Authorization: Bearer` userinfo 요청 |
| RFC 7517 JWK | 공개키 포맷 | JWKS 응답 |
| RFC 7519 JWT | JWT 규약 | `id_token` 기본 형식 |

레퍼런스:
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0-18.html)
- [OpenID Connect Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html)
- [RFC 6749](https://www.rfc-editor.org/rfc/rfc6749.html)
- [RFC 7636](https://www.rfc-editor.org/rfc/rfc7636.html)
- [RFC 6750](https://www.rfc-editor.org/rfc/rfc6750.html)
- [RFC 7517](https://www.rfc-editor.org/rfc/rfc7517.html)
- [RFC 7519](https://www.rfc-editor.org/rfc/rfc7519.html)

## 2. 2차 기준: Google 호환 문서

Google은 표준 OIDC 위에서 자신들의 endpoint와 usage pattern을 제공한다.
이 프로젝트는 그 중 `앱의 Google 로그인 테스트`에 직접 필요한 부분만 맞춘다.

레퍼런스:
- [Google OpenID Connect](https://developers.google.com/identity/openid-connect/openid-connect)

Google 호환 항목:
- authorize 경로를 `/o/oauth2/v2/auth`로 제공
- userinfo 경로를 `/v1/userinfo`로 제공
- jwks 경로를 `/oauth2/v3/certs`로 제공
- `email`, `email_verified`, `name`, `given_name`, `family_name` 중심 claims 제공
- `login_hint`, `nonce`, PKCE, discovery를 통한 통합 테스트 지원

## 3. 이 프로젝트가 목표로 하는 호환 수준

```text
목표
[내 앱 / oauth2-proxy / 테스트 대상]
          |
          v
[mock-google-oidc]
          |
          +-> Google 로그인처럼 redirect
          +-> 표준 OIDC 클라이언트가 discovery/JWKS로 검증 가능
          `-> 로컬에서 빠르게 로그인 테스트 가능
```

한 줄 정의:

```text
"Google 로그인 테스트를 위한 Google-compatible OIDC mock"
```

## 4. 비목표

이 프로젝트는 아래를 목표로 하지 않는다.

- 실제 Google 계정 인증
- 범용 production-grade OpenID Provider
- 동적 클라이언트 등록
- refresh token 지원
- 영속 세션/영속 스토리지
- 엄격한 `client_secret` 값 검증

## 5. 문서 읽는 순서

```text
001-overview.md
   -> 002-reference-specs.md
      -> 003-endpoints.md
         -> 004-google-compatibility.md
            -> 005-flow.md
               -> 006-conformance-boundary.md
                  -> 007-development.md
```
