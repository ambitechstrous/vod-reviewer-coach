package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/client"
	"github.com/ambitechstrous/vod-reviewer-coach/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type IHttpHandler interface {
	// SetUpRouter sets up the HTTP routes for this handler.
	SetUpRouter() *http.Server
	// Start starts the HTTP server. This is separate from SetUpRouter to allow for graceful shutdown.
	Start()
	// Analyze is the endpoint for handling the analyze route.
	Analyze(w http.ResponseWriter, r *http.Request)
	// Login issues a session token for an email.
	Login(w http.ResponseWriter, r *http.Request)
	// Verify confirms the caller's session token is still valid.
	Verify(w http.ResponseWriter, r *http.Request)
	// CreateUploadSession starts a multipart upload and returns presigned part URLs.
	CreateUploadSession(w http.ResponseWriter, r *http.Request)
	// CompleteUpload finalizes a multipart upload once all parts are uploaded.
	CompleteUpload(w http.ResponseWriter, r *http.Request)
	// AbortUpload cancels an in-progress multipart upload.
	AbortUpload(w http.ResponseWriter, r *http.Request)
	// GetUserVideos gets a list view of all videos for a given user
	GetUserVideos(w http.ResponseWriter, r *http.Request)
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
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins(),
		AllowedMethods:   []string{http.MethodGet, http.MethodPost},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Post("/analyze", h.Analyze)

	r.Post("/auth/login", h.Login)

	// TODO: Should we require auth for ALL Requests, or keep as group due to circular issue with login endpoint?
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/auth/verify", h.Verify)
		r.Post("/uploads", h.CreateUploadSession)
		r.Post("/uploads/complete", h.CompleteUpload)
		r.Post("/uploads/abort", h.AbortUpload)
		r.Get("/videos", h.GetUserVideos)
	})

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

// allowedOrigins returns the browser origins allowed to call this API,
// read as a comma-separated list from CORS_ALLOWED_ORIGINS (e.g. the
// deployed frontend's URL), falling back to the local Vite dev server.
func allowedOrigins() []string {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if raw == "" {
		return []string{"http://localhost:5173"}
	}

	origins := strings.Split(raw, ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}
	return origins
}

func (h *HttpHandler) Start() {
	fmt.Printf("analyzer listening on %s\n", h.srv.Addr)
	if err := h.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "analyzer: %v\n", err)
		os.Exit(1)
	}
}
