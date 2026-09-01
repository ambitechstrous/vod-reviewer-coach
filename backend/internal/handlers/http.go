package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/client"
	"github.com/ambitechstrous/vod-reviewer-coach/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type IHttpHandler interface {
	// SetUpRouter sets up the HTTP routes for this handler.
	SetUpRouter() *http.Server
	// Start starts the HTTP server. This is separate from SetUpRouter to allow for graceful shutdown.
	Start()
	// Analyze is the endpoint for handling the analyze route.
	Analyze(w http.ResponseWriter, r *http.Request)
	// CreateUploadSession starts a multipart upload and returns presigned part URLs.
	CreateUploadSession(w http.ResponseWriter, r *http.Request)
	// CompleteUpload finalizes a multipart upload once all parts are uploaded.
	CompleteUpload(w http.ResponseWriter, r *http.Request)
	// AbortUpload cancels an in-progress multipart upload.
	AbortUpload(w http.ResponseWriter, r *http.Request)
}

type HttpHandler struct {
	geminiClient *client.GeminiClient
	s3Client     *storage.S3Client
	srv          *http.Server
}

func NewHttpHandler(ctx context.Context) (IHttpHandler, error) {
	geminiClient, err := client.NewGeminiClient(ctx)
	if err != nil {
		return nil, err
	}
	s3Client, err := storage.NewS3Client(ctx, "user-vods")
	if err != nil {
		return nil, err
	}
	return &HttpHandler{
		geminiClient: geminiClient,
		s3Client:     s3Client,
	}, nil
}

func (h *HttpHandler) SetUpRouter() *http.Server {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Post("/analyze", h.Analyze)

	r.Post("/uploads", h.CreateUploadSession)
	r.Post("/uploads/complete", h.CompleteUpload)
	r.Post("/uploads/abort", h.AbortUpload)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	h.srv = &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}
	return h.srv
}

func (h *HttpHandler) Start() {
	fmt.Printf("analyzer listening on %s\n", h.srv.Addr)
	if err := h.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "analyzer: %v\n", err)
		os.Exit(1)
	}
}
