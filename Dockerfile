FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=$(cat VERSION 2>/dev/null || echo dev)" -o /mock-google-oidc .

FROM scratch
COPY --from=builder /mock-google-oidc /mock-google-oidc
ENTRYPOINT ["/mock-google-oidc"]
