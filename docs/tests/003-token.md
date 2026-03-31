# Test Spec: Token Endpoint

**엔드포인트**: `POST /token`
**준거 표준**: OIDC Core 1.0 §3.1.3, RFC 6749 §4.1.3, RFC 7636 §4.6

---

## 순수 함수 테스트

### T-TOK-V01: ValidateCodeTTL

| createdAt | now | TTL | 기대 |
|-----------|-----|-----|------|
| 12:00 | 12:05 | 10분 | valid (5분 경과) |
| 12:00 | 12:11 | 10분 | expired (11분 경과) |
| 12:00 | 12:10 | 10분 | expired (경계값, >= TTL) |
| 12:00 | 12:00 | 10분 | valid (방금 생성) |

### T-TOK-V02: MatchRedirectURI

| stored | provided | 기대 |
|--------|----------|------|
| `http://localhost/cb` | `http://localhost/cb` | match |
| `http://localhost/cb` | `http://localhost/other` | no match |
| `http://localhost/cb` | `http://localhost/cb/` | no match (trailing slash) |
| `http://localhost/cb` | `https://localhost/cb` | no match (scheme) |
| `http://localhost/cb` | `http://localhost/cb?x=1` | no match (query) |
| `http://localhost/Cb` | `http://localhost/cb` | no match (case) |
| `""` | `""` | match (둘 다 비어있음) |
| `""` | `http://localhost/cb` | match (stored 비어있으면 skip) |

### T-TOK-V03: SplitName

| 입력 | given_name | family_name |
|------|------------|-------------|
| `"Alice Kim"` | `"Alice"` | `"Kim"` |
| `"Alice"` | `"Alice"` | `""` |
| `"Alice Bob Kim"` | `"Alice"` | `"Bob Kim"` |
| `""` | `""` | `""` |

---

## 핸들러 테스트

### T-TOK-01: 정상 토큰 교환

- **요청**: POST code={valid}, client_id=app1, client_secret=secret, redirect_uri=..., grant_type=authorization_code
- **기대**: 200
- **검증**:
  - `access_token` 존재, `ya29.` prefix
  - `id_token` 존재, 유효한 JWT (3파트)
  - `token_type` == `"Bearer"`
  - `expires_in` == 3920
  - `scope` 존재

### T-TOK-02: Cache-Control 헤더

- **기대**: 200 응답에 아래 헤더 포함
  - `Cache-Control: no-store`
  - `Pragma: no-cache`
- **스펙**: Core §3.1.3.3

### T-TOK-03: redirect_uri 불일치

- **설정**: auth 시 redirect_uri=`http://example.com/cb`
- **요청**: token 시 redirect_uri=`http://example.com/other`
- **기대**: 400, `error=invalid_grant`
- **스펙**: RFC 6749 §4.1.3

### T-TOK-04: 존재하지 않는 코드

- **요청**: code=nonexistent
- **기대**: 400, `error=invalid_grant`

### T-TOK-05: 빈 코드

- **요청**: code 파라미터 없음
- **기대**: 400, `error=invalid_grant`

### T-TOK-06: 코드 재사용 거부

- **1차 요청**: 성공 (200)
- **2차 요청**: 동일 코드로 재시도
- **기대**: 400, `error=invalid_grant`
- **스펙**: RFC 6749 §4.1.2

### T-TOK-07: 잘못된 grant_type

- **요청**: grant_type=refresh_token
- **기대**: 400, `error=unsupported_grant_type`

### T-TOK-08: token_error 모드

- **설정**: response_mode=token_error로 인가
- **기대**: 500, `error=server_error`

### T-TOK-09: client_id 불일치

- **설정**: auth 시 client_id=app1
- **요청**: token 시 client_id=app2
- **기대**: 400, `error=invalid_grant`

### T-TOK-10: client_secret_basic 인증

- **요청**: Authorization: Basic base64(app1:secret), body에 client_id/secret 없음
- **기대**: 200 (정상 교환)
- **스펙**: RFC 6749 §2.3.1

---

## PKCE 테스트

### T-TOK-P01: S256 정상

- **설정**: code_challenge=BASE64URL(SHA256(verifier)), method=S256
- **요청**: code_verifier={verifier}
- **기대**: 200

### T-TOK-P02: S256 잘못된 verifier

- **요청**: code_verifier=wrong
- **기대**: 400, `error=invalid_grant`

### T-TOK-P03: verifier 누락

- **설정**: code_challenge 있음
- **요청**: code_verifier 없음
- **기대**: 400, `error=invalid_grant`

### T-TOK-P04: plain 정상

- **설정**: code_challenge={verifier}, method=plain
- **요청**: code_verifier={verifier}
- **기대**: 200

### T-TOK-P05: PKCE 미사용

- **설정**: code_challenge 없음
- **요청**: code_verifier 없음
- **기대**: 200 (PKCE 선택사항)

### T-TOK-P06: PKCE 실패 시 코드 미소모

- **1차**: 잘못된 verifier → 400
- **2차**: 올바른 verifier → 200 (코드 여전히 유효)
- **스펙**: RFC 7636 best practice

---

## ID Token Claims 테스트

### T-TOK-C01: 필수 claims 존재

- **검증**: `iss`, `sub`, `aud`, `exp`, `iat` 모두 존재
- **스펙**: Core §2

### T-TOK-C02: nonce 포함

- **설정**: 인가 시 nonce=nonce123
- **검증**: id_token에 `nonce=nonce123`
- **스펙**: Core §3.1.3.7

### T-TOK-C03: azp claim

- **검증**: `azp` == `client_id`

### T-TOK-C04: 이름 분리

- **설정**: name="Claims User"
- **검증**: `given_name=Claims`, `family_name=User`

### T-TOK-C05: iss 값

- **검증**: `iss` == discovery의 `issuer`
