# Test Specifications

mock-google-oidc의 `Google 로그인 테스트용 OIDC 계약`을 보장하기 위한 테스트 명세.

## 테스트 계층

```
Layer 1: 순수 함수 테스트 (validate_test.go)
  └─ HTTP 컨텍스트 없이 검증 로직만 테스트
  └─ ValidateCodeTTL, MatchRedirectURI, ValidateResponseType,
     RequireOpenIDScope, ValidatePrompt, SplitName

Layer 2: 핸들러 테스트 (handler_test.go)
  └─ httptest로 각 엔드포인트의 요청/응답 검증
  └─ 입력 검증, 에러 응답, 정상 응답 형식

Layer 3: 통합 플로우 테스트 (flow_test.go)
  └─ 전체 인증 플로우 엔드투엔드 검증
  └─ 정상, 에러, PKCE 시나리오

Layer 4: Conformance 테스트 (conformance_test.go)
  └─ coreos/go-oidc 표준 클라이언트로 provider 검증
  └─ "Google-compatible mock이 표준 클라이언트와도 호환됨" 증명
```

## 상위 문서와의 관계

```text
docs/002-reference-specs.md
   -> 어떤 외부 스펙을 기준으로 삼는가

docs/003-endpoints.md
   -> 무엇을 endpoint contract로 테스트하는가

docs/004-google-compatibility.md
   -> 왜 Google 스타일 경로/동작을 테스트하는가

docs/006-conformance-boundary.md
   -> MUST/SHOULD/비준수 항목 중 무엇을 테스트로 증명하는가
```

## 테스트 문서 목록

| 파일 | 대상 | 테스트 수 |
|------|------|----------|
| [001-discovery.md](001-discovery.md) | Discovery Endpoint | 5 |
| [002-authorization.md](002-authorization.md) | Authorization Endpoint | 15 |
| [003-token.md](003-token.md) | Token Endpoint + PKCE + ID Token Claims | 21 |
| [004-userinfo.md](004-userinfo.md) | UserInfo Endpoint | 6 |
| [005-jwks.md](005-jwks.md) | JWKS Endpoint | 5 |
| [006-conformance.md](006-conformance.md) | coreos/go-oidc Conformance | 9 |
| [007-flow.md](007-flow.md) | Full Flow Integration | 9 |

## 테스트 ID 체계

```
T-{영역}-{번호}     핸들러/순수함수 테스트
T-{영역}-V{번호}    순수 함수(Validate) 테스트
T-{영역}-P{번호}    PKCE 테스트
T-{영역}-C{번호}    Claims 테스트
T-{영역}-G{번호}    GET 핸들러 테스트
C{번호}             Conformance 테스트
C-FLOW-{번호}       Conformance 통합 플로우
```

| 영역 코드 | 대상 |
|-----------|------|
| DISC | Discovery |
| AUTH | Authorization |
| TOK | Token |
| UI | UserInfo |
| JWK | JWKS |
| FLOW | Full Flow |

## 실행

```bash
# 전체 테스트
go test ./internal/oidc/ -v

# 순수 함수만
go test ./internal/oidc/ -run "TestValidate|TestMatch|TestRequire|TestSplitName" -v

# 핸들러만
go test ./internal/oidc/ -run "TestDiscovery|TestAuthorize|TestToken|TestUserinfo|TestCerts|TestHealth" -v

# 통합 플로우만
go test ./internal/oidc/ -run "TestFullFlow" -v

# Conformance만
go test ./internal/oidc/ -run "TestConformance" -v
```

주의:

```text
conformance_test.go는 외부 OIDC client library 의존성이 필요하다.
즉 "전체 테스트가 돌아야 한다"는 목표에는
go.mod가 conformance 의존성을 포함해야 한다는 전제가 있다.
```

## 스펙 ↔ 테스트 매핑

| 스펙 (006-conformance-boundary.md) | 테스트 코드 | 상태 |
|---------------------------|------------|------|
| Discovery MUST 필드 | T-DISC-01, T-DISC-02 | ✅ 구현됨 |
| response_type 검증 | T-AUTH-V01, T-AUTH-G05~G06 | ✅ 구현됨 |
| openid scope 검증 | T-AUTH-V02, T-AUTH-G07 | ✅ 구현됨 |
| prompt=none 처리 | T-AUTH-V03, T-AUTH-G08 | ✅ 구현됨 |
| Code TTL 10분 | T-TOK-V01 | ✅ 구현됨 |
| redirect_uri 매칭 | T-TOK-V02, T-TOK-03 | ✅ 구현됨 |
| Code single-use | T-TOK-06 | ✅ 구현됨 |
| PKCE S256/plain | T-TOK-P01~P06 | ✅ 구현됨 |
| Cache-Control 헤더 | T-TOK-02, T-JWK-03 | ✅ 구현됨 |
| id_token claims | T-TOK-C01~C05 | ✅ 구현됨 |
| UserInfo sub 일관성 | T-UI-05 | ✅ 구현됨 |
| given/family name 분리 | T-TOK-V03, T-FLOW-09 | ✅ 구현됨 |
| Conformance C1~C7 | C1~C7, C-FLOW-01~02 | ✅ 구현됨 |
