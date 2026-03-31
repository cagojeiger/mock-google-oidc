# Spec 004: Google 호환 범위

이 문서는 mock-google-oidc가 `무엇을 Google과 맞추고`, `무엇은 단순화하는지`를 설명한다.

## 큰 원칙

```text
[프로토콜]
  가능한 한 표준 OIDC/OAuth를 따름

[경로 / 사용 패턴]
  가능한 한 Google 스타일을 따름

[로그인 UX]
  테스트 편의상 대폭 단순화
```

## 1. Google과 맞추는 것

| 항목 | 현재 정책 |
| --- | --- |
| authorize 경로 | `/o/oauth2/v2/auth` |
| token 경로 | `/token` |
| userinfo 경로 | `/v1/userinfo` |
| JWKS 경로 | `/oauth2/v3/certs` |
| discovery 제공 | 예 |
| Authorization Code Flow | 예 |
| PKCE | `S256`, `plain` |
| `id_token` 서명 | RS256 |
| `userinfo` claims | `sub`, `email`, `email_verified`, `name`, `given_name`, `family_name`, `picture` |
| `login_hint` | 기본 email에 반영 |
| `nonce` | `id_token`에 반영 |

## 2. Google과 다르게 단순화하는 것

| 항목 | 실제 Google | mock-google-oidc |
| --- | --- | --- |
| 사용자 인증 | 실제 계정/세션 | 이메일 + 이름 입력 |
| 비밀번호 | 필요 | 없음 |
| 동의 화면 | 경우에 따라 있음 | 없음 |
| client 등록 | Google Cloud Console | 없음 |
| `client_secret` 검증 | 실제 값 검증 | 존재 여부만 체크 |
| TLS | HTTPS | 로컬 HTTP 허용 |
| refresh token | 지원 가능 | 미지원 |

## 3. 로그인 화면 정책

이 프로젝트의 유일한 사용자 대면 화면은 `GET /o/oauth2/v2/auth`에서 렌더링되는 mock 로그인 화면이다.

```text
+----------------------------------------------------------+
|  Test IDP                                                |
|----------------------------------------------------------|
|  Sign in to continue                                     |
|                                                          |
|  Email : [ alice@example.com                          ]  |
|  Name  : [ Alice                                      ]  |
|                                                          |
|                    [ Login ]                             |
|                                                          |
|  ▸ Advanced (Response Mode)                              |
|    (•) Normal  ( ) Deny  ( ) Token Error                 |
|    ( ) Userinfo Error                                    |
|                                                          |
|  client_id: my-app                                       |
|  redirect_uri: http://localhost:8080/callback            |
|  scope: openid email profile                             |
|  state: abc123                                           |
+----------------------------------------------------------+
```

이 화면에서 Google과의 차이는 명확하다.

```text
Google:
  계정 선택 -> 비밀번호 -> 동의

mock-google-oidc:
  이메일/이름 입력 -> Login
```

## 4. mock 전용 기능

아래 항목은 Google 자체보다 `테스트 도구`로서의 편의를 위한 기능이다.

| 기능 | 목적 |
| --- | --- |
| `response_mode=deny` | 사용자가 권한을 거부한 상황 테스트 |
| `response_mode=token_error` | token endpoint 실패 처리 테스트 |
| `response_mode=userinfo_error` | userinfo endpoint 실패 처리 테스트 |
| deterministic `sub` | 같은 이메일로 재로그인 시 같은 사용자로 인식되도록 보장 |

## 5. 호환성 판단 기준

이 프로젝트에서 "Google 호환"은 아래 뜻으로 사용한다.

```text
[앱]
  Google 로그인용 OIDC 설정
     |
     v
[mock-google-oidc]
  -> 비슷한 endpoint
  -> 비슷한 claims
  -> 같은 redirect/callback 흐름
  -> 표준 클라이언트 검증 가능
```

즉,
- 실제 Google 계정으로 로그인되는 것은 아니다.
- 하지만 앱 입장에서는 "Google 로그인과 유사한 OIDC 플로우"를 테스트할 수 있어야 한다.
