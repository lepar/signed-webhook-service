package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"kii.com/internal/application/usecase"
	"kii.com/internal/domain/entity"
	"kii.com/internal/domain/port"
	"kii.com/internal/infrastructure/logger"
)

// Handler holds HTTP handlers and their dependencies
type Handler struct {
	processWebhookUseCase *usecase.ProcessWebhookUseCase
	getBalanceUseCase     *usecase.GetBalanceUseCase
	validator             port.WebhookValidator
	logger                logger.Logger
}

// NewHandler creates a new HTTP handler
func NewHandler(
	processWebhookUseCase *usecase.ProcessWebhookUseCase,
	getBalanceUseCase *usecase.GetBalanceUseCase,
	validator port.WebhookValidator,
	logger logger.Logger,
) *Handler {
	return &Handler{
		processWebhookUseCase: processWebhookUseCase,
		getBalanceUseCase:     getBalanceUseCase,
		validator:             validator,
		logger:                logger,
	}
}

// HandleWebhook handles POST /webhook requests
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestLogger := ctx.Value("logger").(logger.Logger)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		requestLogger.LogError(ctx, "Failed to read request body", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Validate webhook signature
	if err := h.validator.ValidateRequest(ctx, r, body); err != nil {
		requestLogger.LogWarning(ctx, "Webhook validation failed", err)
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusUnauthorized)
		return
	}

	// Parse JSON body
	var webhookReq entity.WebhookRequest
	if err := json.Unmarshal(body, &webhookReq); err != nil {
		requestLogger.LogError(ctx, "Failed to parse JSON body", err)
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Execute use case
	req := usecase.ProcessWebhookRequest{
		WebhookRequest: &webhookReq,
		HTTPRequest: &httpRequestAdapter{
			header: r.Header,
			body:   body,
		},
	}

	if err := h.processWebhookUseCase.Execute(ctx, req); err != nil {
		requestLogger.LogError(ctx, "Failed to process webhook", err)
		http.Error(w, fmt.Sprintf("Failed to process webhook: %v", err), http.StatusInternalServerError)
		return
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	requestLogger.LogInfo(ctx, "Webhook processed successfully",
		"user", webhookReq.User,
		"asset", webhookReq.Asset,
		"amount", webhookReq.Amount)
}

// HandleBalance handles GET /balance/{user} requests
func (h *Handler) HandleBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestLogger := ctx.Value("logger").(logger.Logger)

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user from path
	path := strings.TrimPrefix(r.URL.Path, "/balance/")
	if path == "" || path == r.URL.Path {
		http.Error(w, "Missing user parameter", http.StatusBadRequest)
		return
	}

	user := path

	// Execute use case
	balance, err := h.getBalanceUseCase.Execute(ctx, user)
	if err != nil {
		requestLogger.LogError(ctx, "Failed to get balance", err)
		http.Error(w, "Failed to get balance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(balance); err != nil {
		requestLogger.LogError(ctx, "Failed to encode balance response", err)
		return
	}

	requestLogger.LogInfo(ctx, "Balance retrieved",
		"user", user)
}

// httpRequestAdapter adapts http.Request to the interface expected by use case
type httpRequestAdapter struct {
	header http.Header
	body   []byte
}

func (a *httpRequestAdapter) Header() map[string][]string {
	return a.header
}

func (a *httpRequestAdapter) Body() []byte {
	return a.body
}

// SetupRoutes sets up all HTTP routes
func (h *Handler) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Apply middleware chain
	webhookHandler := RequestIDMiddleware(
		LoggingMiddleware(h.HandleWebhook, h.logger),
		h.logger,
	)
	balanceHandler := RequestIDMiddleware(
		LoggingMiddleware(h.HandleBalance, h.logger),
		h.logger,
	)

	mux.HandleFunc("/webhook", webhookHandler)
	mux.HandleFunc("/balance/", balanceHandler)

	return mux
}
