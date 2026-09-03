package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/topi314/campfire-export/server/cache"
	"github.com/topi314/campfire-export/server/campfire"
	"github.com/topi314/campfire-export/server/web"
)

type Server struct {
	cfg    Config
	client *campfire.Client
	cache  *cache.Cache
	http   *http.Server
}

func New(cfg Config) (*Server, error) {
	s := &Server{
		cfg:    cfg,
		client: campfire.New(cfg.Campfire),
		cache:  cache.New(cfg.Cache.TTL),
	}
	handler, err := s.Handler()
	if err != nil {
		return nil, err
	}
	s.http = &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s, nil
}

func (s *Server) Start() {
	go func() {
		slog.Info("listening", slog.String("addr", s.cfg.Server.Addr))
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", slog.Any("err", err))
		}
	}()
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.http.Shutdown(ctx); err != nil {
		slog.Error("error while shutting down server", slog.Any("err", err))
	}
}

func (s *Server) Handler() (http.Handler, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/config", s.clientConfig)
	mux.HandleFunc("GET /api/pois", s.getPOIs)
	mux.HandleFunc("GET /api/places", s.searchPlaces)
	mux.HandleFunc("POST /api/export", s.exportKMZ)

	spa, err := web.Handler()
	if err != nil {
		return nil, err
	}
	mux.Handle("/", spa)
	return s.cors(mux), nil
}
