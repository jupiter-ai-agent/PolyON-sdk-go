# PolyON SDK for Go

PolyON Platform(PP) 모듈 개발을 위한 Go SDK.

PRC(Platform Resource Claim)로 프로비저닝된 자원에 쉽게 접근하고,
PP 표준 패턴(OIDC 인증, Health Check 등)을 자동화합니다.

## 설치

```bash
go get github.com/jupiter-ai-agent/PolyON-sdk-go
```

## 모듈 구성

| 패키지 | 용도 |
|--------|------|
| `polyon` | Config 자동 파싱 (PRC 환경변수) |
| `polyon/auth` | OIDC 토큰 검증 미들웨어 |
| `polyon/health` | 표준 /health 엔드포인트 |
| `polyon/storage` | S3(RustFS) 파일 업로드/다운로드 |
| `polyon/directory` | LDAP 사용자/그룹 조회 |
| `polyon/search` | OpenSearch 인덱싱/검색 |
| `polyon/cache` | Redis 접근 |

## Quick Start

```go
package main

import (
    "log"
    "net/http"

    polyon "github.com/jupiter-ai-agent/PolyON-sdk-go"
    "github.com/jupiter-ai-agent/PolyON-sdk-go/auth"
    "github.com/jupiter-ai-agent/PolyON-sdk-go/health"
)

func main() {
    // PRC 환경변수 자동 로드
    cfg := polyon.MustLoad()

    // OIDC 미들웨어
    oidc := auth.NewVerifier(cfg.Auth)

    mux := http.NewServeMux()
    mux.Handle("/health", health.Handler())
    mux.Handle("/api/", oidc.Middleware(apiHandler()))

    log.Printf("Starting on :8080 (issuer=%s)", cfg.Auth.Issuer)
    http.ListenAndServe(":8080", mux)
}
```

## PRC 환경변수 매핑

SDK는 PRC가 주입하는 환경변수를 자동으로 파싱합니다.

| 환경변수 | Config 필드 |
|---------|------------|
| `OIDC_ISSUER` | `cfg.Auth.Issuer` |
| `OIDC_CLIENT_ID` | `cfg.Auth.ClientID` |
| `OIDC_CLIENT_SECRET` | `cfg.Auth.ClientSecret` |
| `OIDC_TOKEN_ENDPOINT` | `cfg.Auth.TokenEndpoint` |
| `OIDC_JWKS_URI` | `cfg.Auth.JWKSURI` |
| `DATABASE_URL` | `cfg.Database.URL` |
| `S3_ENDPOINT` | `cfg.Storage.Endpoint` |
| `S3_BUCKET` | `cfg.Storage.Bucket` |
| `S3_ACCESS_KEY` | `cfg.Storage.AccessKey` |
| `S3_SECRET_KEY` | `cfg.Storage.SecretKey` |

## 규격

- [PRC Provider Reference](https://github.com/jupiter-ai-agent/PolyON-platform/blob/main/docs/prc-provider-reference.md)
- [Module Spec](https://github.com/jupiter-ai-agent/PolyON-platform/blob/main/docs/module-spec.md)

## 라이선스

MIT — Triangle.s
