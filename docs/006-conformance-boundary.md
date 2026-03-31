# Spec 006: 준수 경계와 테스트 기준

이 문서는 `무엇을 반드시 보장할지`, `무엇은 의도적으로 단순화할지`, `무엇을 테스트로 증명할지`를 정의한다.

## 전체 그림

```text
[표준 OIDC/OAuth]
        |
        v
  핵심 계약 준수
        |
        +-> discovery
        +-> authorization code flow
        +-> PKCE
        +-> id_token
        +-> userinfo
        +-> jwks
        |
        v
[Google 호환]
        |
        v
  경로/claim/플로우 모양 조정
        |
        v
[mock 제약]
  로컬 테스트를 위해 일부 의도적 비준수
```

## 1. MUST 보장 항목

### Discovery
- discovery 문서가 유효한 JSON이어야 한다.
- 표준 클라이언트가 issuer, authorize, token, userinfo, jwks endpoint를 찾을 수 있어야 한다.

### Authorization
- `response_type=code`만 허용한다.
- `scope`에는 `openid`가 포함되어야 한다.
- `state`를 그대로 돌려준다.
- `prompt=none`은 `login_required`로 거부한다.

### Token
- `grant_type=authorization_code`만 허용한다.
- code는 1회용이다.
- code는 10분 이내에만 유효하다.
- `redirect_uri`는 authorize 요청과 exact match여야 한다.
- PKCE `plain`, `S256`를 지원한다.
- PKCE 실패 시 code를 소비하지 않는다.

### ID Token / JWKS
- `id_token`은 RS256으로 서명한다.
- `iss`, `sub`, `aud`, `exp`, `iat`를 포함한다.
- JWKS에서 공개키를 제공한다.
- OIDC 클라이언트가 JWKS로 서명을 검증할 수 있어야 한다.

### UserInfo
- `Authorization: Bearer`를 요구한다.
- `userinfo.sub`는 `id_token.sub`와 일치해야 한다.
- `email`, `email_verified`, `name` 계열 claim을 반환한다.

## 2. SHOULD 보장 항목

- `client_secret_post`와 `client_secret_basic` 둘 다 지원
- `login_hint`를 email 기본값에 반영
- `nonce`를 `id_token`에 반영
- Google 스타일 경로 유지
- `oauth2-proxy` 같은 실제 RP와 수동 통합 확인 가능

## 3. 의도적 비준수

| 항목 | 표준 기대 | 현재 정책 | 이유 |
| --- | --- | --- | --- |
| TLS | HTTPS 전제 | 로컬 HTTP 허용 | 테스트 편의 |
| 실제 사용자 인증 | 실제 계정/세션 | 이메일+이름 입력 | mock 특성 |
| `client_secret` 값 검증 | 실제 비밀값 검증 | 존재 여부만 확인 | 설정 단순화 |
| client 사전 등록 | 일반적으로 필요 | 미지원 | 빠른 통합 테스트 |
| redirect URI 사전 등록 | 일반적으로 필요 | 미지원 | 빠른 통합 테스트 |
| refresh token | 구현 가능 | 미지원 | 목표 범위 밖 |

## 4. 테스트로 증명해야 하는 것

```text
Layer 1: 순수 함수
  -> TTL / redirect URI / prompt / scope / response_type

Layer 2: 핸들러 계약
  -> discovery / authorize / token / userinfo / jwks / health

Layer 3: 전체 플로우
  -> authorize -> token -> userinfo
  -> PKCE
  -> error modes

Layer 4: conformance
  -> coreos/go-oidc
  -> JWKS 기반 verify
  -> 표준 OIDC client compatibility
```

핵심 증명 기준:
- `coreos/go-oidc`가 discovery를 읽을 수 있어야 한다.
- `Verifier.Verify`로 `id_token` 검증이 성공해야 한다.
- `UserInfo` 호출이 성공해야 한다.
- `oauth2-proxy` 수동 통합이 가능해야 한다.

## 5. 테스트 항목 체크리스트

### Discovery
- [ ] issuer와 endpoint URL이 올바르다
- [ ] 지원 capability metadata가 광고된다

### Authorization
- [ ] `response_type=token` 거부
- [ ] `scope`에서 `openid` 누락 시 거부
- [ ] `prompt=none` 거부
- [ ] `state`가 redirect에 유지

### Token
- [ ] 정상 code redeem
- [ ] invalid code 거부
- [ ] code 재사용 거부
- [ ] redirect URI mismatch 거부
- [ ] PKCE `S256` 성공/실패
- [ ] PKCE `plain` 성공
- [ ] cache-control 헤더 존재

### ID Token / UserInfo
- [ ] RS256 검증 성공
- [ ] `nonce` 반영
- [ ] deterministic `sub`
- [ ] `userinfo.sub == id_token.sub`

### Compatibility
- [ ] Google 스타일 경로로 flow 수행 가능
- [ ] `oauth2-proxy`와 통합 가능

## 6. 에러 응답 원칙

Authorization 단계는 redirect 에러를 사용한다.

```text
{redirect_uri}?error={error_code}&error_description={desc}&state={state}
```

Token/UserInfo 단계는 JSON 에러를 사용한다.

```json
{
  "error": "{error_code}",
  "error_description": "{description}"
}
```

## 7. Conformance 검증 매트릭스

`coreos/go-oidc`로 아래를 증명하는 것을 목표로 한다.

| ID | 검증 항목 | 기대 결과 |
| --- | --- | --- |
| C1 | discovery 파싱 | `oidc.NewProvider()` 성공 |
| C2 | JWKS 로딩 | verifier 생성 가능 |
| C3 | `id_token` 서명 검증 | `Verify()` 성공 |
| C4 | claims 파싱 | 필수 claim 접근 가능 |
| C5 | sub 일관성 | `id_token.sub == userinfo.sub` |
| C6 | userinfo 호출 | `UserInfo()` 성공 |
| C7 | 만료 토큰 거부 | expired token 검증 실패 |
