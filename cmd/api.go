package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	handler "github.com/antoniobt12062002/pao-de-queijo/internal/handler/http"
	_ "github.com/antoniobt12062002/pao-de-queijo/docs/swagger"
	httpSwagger "github.com/swaggo/http-swagger"
)

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("all good"))
	})

	r.Route("/v1", func(r chi.Router) {
		r.Post("/users", app.userHandler.Register)
		r.Post("/auth/login", app.authHandler.Login)
		r.Get("/auth/github", app.authHandler.GitHubLogin)
		r.Get("/auth/github/callback", app.authHandler.GitHubCallback)
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	return r
}

func (app *application) run(h http.Handler) error {
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      h,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	slog.Info("starting server", "addr", app.config.addr)
	return srv.ListenAndServe()
}

type application struct {
	config      config
	userHandler *handler.UserHandler
	authHandler *handler.AuthHandler
}

type config struct {
	addr string
	db   dbConfig
}

type dbConfig struct {
	dsn string
}
