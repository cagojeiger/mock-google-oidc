# Test Spec: Authorization Endpoint

**엔드포인트**: `GET /o/oauth2/v2/auth`, `POST /o/oauth2/v2/auth`
**준거 표준**: OIDC Core 1.0 §3.1.2, RFC 6749 §4.1.1

---

## 순수 함수 테스트

### T-AUTH-V01: ValidateResponseType

| 입력 | 기대 결과 |
|------|----------|
| `"code"` | 통과 |
| `"token"` | 에러 (implicit 미지원) |
| `"id_token"` | 에러 |
| `""` | 에러 (필수 파라미터) |
| `"code id_token"` | 에러 (hybrid 미지원) |

### T-AUTH-V02: RequireOpenIDScope

| 입력 | 기대 결과 |
|------|----------|
| `"openid email profile"` | 통과 |
| `"openid"` | 통과 |
| `"email openid profile"` | 통과 (위치 무관) |
| `"email profile"` | 에러 (openid 누락) |
| `""` | 에러 |
| `"notopenid email"` | 에러 (부분 문자열 불일치) |

### T-AUTH-V03: ValidatePrompt

| 입력 | 기대 결과 |
|------|----------|
| `""` | 통과 (기본 동작) |
| `"consent"` | 통과 |
| `"login"` | 통과 |
| `"select_account"` | 통과 |
| `"none"` | 에러 `login_required` (mock은 항상 로그인 필요) |
| `"none login"` | 에러 (none은 다른 값과 조합 불가, Core §3.1.2.1) |

---

## 핸들러 테스트 — GET (로그인 폼)

### T-AUTH-G01: 정상 요청

- **요청**: `GET /o/oauth2/v2/auth?redirect_uri=...&state=abc&client_id=app1&scope=openid&response_type=code`
- **기대**: 200, HTML 로그인 폼 반환
- **검증**: 폼에 기본 이메일 `alice@example.com` 포함

### T-AUTH-G02: login_hint 적용

- **요청**: `...&login_hint=bob@test.com`
- **기대**: 폼에 `bob@test.com` 표시

### T-AUTH-G03: redirect_uri 누락

- **요청**: `GET /o/oauth2/v2/auth?state=abc`
- **기대**: 400

### T-AUTH-G04: state 누락

- **요청**: `GET /o/oauth2/v2/auth?redirect_uri=...`
- **기대**: 400

### T-AUTH-G05: response_type 누락

- **요청**: `...&scope=openid` (response_type 없음)
- **기대**: 302 redirect → `error=unsupported_response_type`

### T-AUTH-G06: response_type=token

- **요청**: `...&response_type=token`
- **기대**: 302 redirect → `error=unsupported_response_type`

### T-AUTH-G07: openid scope 누락

- **요청**: `...&scope=email&response_type=code`
- **기대**: 302 redirect → `error=invalid_scope`

### T-AUTH-G08: prompt=none

- **요청**: `...&prompt=none`
- **기대**: 302 redirect → `error=login_required`

---

## 핸들러 테스트 — POST (코드 발급)

### T-AUTH-P01: 정상 로그인

- **요청**: POST email=alice@example.com, name=Alice, state=abc, redirect_uri=...
- **기대**: 302 redirect → `?code={code}&state=abc`
- **검증**: code 비어있지 않음, state 보존

### T-AUTH-P02: deny 모드

- **요청**: POST ...response_mode=deny
- **기대**: 302 redirect → `?error=access_denied&state=abc`
- **검증**: code 없음

### T-AUTH-P03: email 누락

- **요청**: POST name=Alice, state=abc (email 없음)
- **기대**: 400

### T-AUTH-P04: name 누락

- **요청**: POST email=alice@example.com, state=abc (name 없음)
- **기대**: 400

### T-AUTH-P05: redirect_uri 저장

- **요청**: POST ...redirect_uri=http://example.com/cb
- **검증**: 저장된 CodeEntry에 RedirectURI가 기록됨 (token endpoint에서 비교용)
