# Spec 001: 프로젝트 개요

## 목적

이 프로젝트의 목적은 `Google 로그인 연동을 로컬과 테스트 환경에서 빠르게 검증할 수 있는 mock OIDC Provider`를 제공하는 것이다.

이 서버는 실제 Google이 아니다.
대신 앱이 기대하는 Google 로그인 흐름을 최대한 비슷하게 재현한다.

## 한 줄 요약

```text
Google 로그인 테스트를 위한 Google-compatible OIDC mock provider.
```

## 전체 위치

```text
[브라우저]
    |
    v
[내 앱 / oauth2-proxy]
    |
    v
[mock-google-oidc]
    |
    +-> Google 스타일 authorize 경로
    +-> token / userinfo / jwks
    +-> password 없는 mock 로그인
```

## 무엇을 해결하나

- 실제 Google 계정 없이 로그인 플로우를 검증하고 싶다
- Google Cloud Console 설정 없이 로컬에서 통합 테스트를 돌리고 싶다
- `oauth2-proxy` 같은 RP가 OIDC client로 제대로 붙는지 확인하고 싶다
- PKCE, `nonce`, `userinfo`, JWKS 검증까지 포함한 흐름을 테스트하고 싶다

## 제품 성격

```text
이 프로젝트는
"범용 production OpenID Provider"가 아니라
"Google 로그인 테스트용 mock provider"다.
```

즉:
- 표준 OIDC/OAuth 문서를 기준으로 핵심 계약을 정의한다
- Google 문서를 기준으로 경로와 사용 패턴을 맞춘다
- 로컬 mock 특성상 일부 항목은 의도적으로 단순화한다

## 설계 원칙

1. Google 로그인 테스트에 필요한 경로와 claims를 제공한다.
2. Authorization Code + PKCE 흐름을 안정적으로 지원한다.
3. OIDC discovery와 JWKS를 통해 표준 클라이언트 검증이 가능해야 한다.
4. 로그인 UX는 이메일+이름 입력만으로 최대한 단순화한다.
5. 외부 서비스나 DB 없이 단독 실행 가능해야 한다.

## 지원 범위

| 항목 | 지원 여부 | 비고 |
| --- | --- | --- |
| Authorization Code Flow | 예 | 핵심 목표 |
| PKCE `S256`, `plain` | 예 | 테스트 중요 항목 |
| discovery | 예 | 표준 client 호환성 |
| JWKS | 예 | RS256 검증용 |
| `id_token` | 예 | RS256 |
| `userinfo` | 예 | email/name 계열 claims |
| `nonce` | 예 | 요청 시 `id_token`에 반영 |
| deterministic `sub` | 예 | 이메일 기반 |
| mock 에러 모드 | 예 | deny/token_error/userinfo_error |
| refresh token | 아니오 | 목표 범위 밖 |
| 실제 Google 계정 인증 | 아니오 | mock |
| 엄격한 client registration | 아니오 | 단순화 |

## 문서 구성

```text
001-overview.md
  프로젝트 목표와 범위

002-reference-specs.md
  어떤 외부 스펙을 기준으로 삼는지

003-endpoints.md
  HTTP 계약

004-google-compatibility.md
  Google과 무엇을 맞추고 무엇을 단순화하는지

005-flow.md
  전체 인증 흐름

006-conformance-boundary.md
  MUST / SHOULD / intentionally non-compliant

007-development.md
  실행, 개발, 수동 검증
```
