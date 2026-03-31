package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/kangheeyong/mock-google-oidc/internal/oidc"
)

var version = "dev"

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	addr := envOr("LISTEN_ADDR", ":8082")
	publicURL := envOr("PUBLIC_URL", "http://localhost:8082")

	keys := oidc.NewKeyPair()
	store := oidc.NewStore()

	mux := http.NewServeMux()
	oidc.RegisterHandlers(mux, publicURL, keys, store, version)

	slog.Info("mock-google-oidc starting", "addr", addr, "publicURL", publicURL, "version", version)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
