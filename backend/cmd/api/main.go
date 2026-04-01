package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/auth"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/buildinfo"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/discovery"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/handlers"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/health"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/middleware"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/session"
)

func main() {
	// Structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	mux := http.NewServeMux()

	// Template routes (do not modify)
	mux.HandleFunc("GET /api/health", health.Handler)
	mux.HandleFunc("GET /api/build", buildinfo.Handler)
	mux.HandleFunc("GET /api/discovery", discovery.Handler(handlers.RegisterDiscoveryLinks))
	mux.HandleFunc("GET /api/session", session.Handler)

	// App routes (managed by add-endpoint.sh)
	handlers.RegisterRoutes(mux)

	// Middleware chain
	handler := middleware.Chain(
		mux,
		middleware.Recovery,
		middleware.Logging,
		middleware.CORS,
		middleware.CorrelationID,
		middleware.TxnLogMiddleware,
		auth.Middleware,
	)

	// Lambda or local server
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		adapter := httpadapter.NewV2(handler)
		lambda.Start(adapter.ProxyWithContext)
	} else {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		slog.Info("Starting server", "port", port)
		if err := http.ListenAndServe(":"+port, handler); err != nil {
			log.Fatal(err)
		}
	}
}
