package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	handler "github.com/antoniobt12062002/pao-de-queijo/internal/handler/http"
	_ "github.com/antoniobt12062002/pao-de-queijo/docs/swagger"
	httpSwagger "github.com/swaggo/http-swagger"
)

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:4173", "https://glowing-syrniki-17cec4.netlify.app"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("all good"))
	})

	r.Route("/v1", func(r chi.Router) {
		// Public routes
		r.Post("/users", app.userHandler.Register)
		r.Post("/auth/login", app.authHandler.Login)
		r.Get("/auth/github", app.authHandler.GitHubLogin)
		r.Get("/auth/github/callback", app.authHandler.GitHubCallback)

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(handler.JWTMiddleware(app.config.jwtSecret))

			r.Get("/users", app.userHandler.List)
			r.With(handler.AdminOnly).Put("/users/{id}/role", app.userHandler.UpdateRole)

			r.Get("/config", app.configHandler.GetAll)
			r.With(handler.AdminOnly).Put("/config", app.configHandler.Update)

			r.Get("/rotation", app.rotationHandler.GetCurrent)
			r.With(handler.AdminOnly).Put("/rotation/order", app.rotationHandler.UpdateOrder)
			r.With(handler.AdminOnly).Post("/rotation/skip", app.rotationHandler.Skip)

			r.Get("/rounds", app.roundHandler.GetAll)
			r.Get("/rounds/today", app.roundHandler.GetToday)
			r.Post("/rounds/{id}/confirm", app.roundHandler.Confirm)
			r.Post("/rounds/{id}/cancel", app.roundHandler.Cancel)
			r.Put("/rounds/{id}/cost", app.roundHandler.SetActualCost)

			r.Post("/rounds/{id}/participate", app.participationHandler.Participate)
			r.Delete("/rounds/{id}/participate", app.participationHandler.Withdraw)
			r.Get("/rounds/{id}/participations", app.participationHandler.GetParticipations)

			r.Post("/devices", app.deviceHandler.Register)
			r.Delete("/devices/{token}", app.deviceHandler.Remove)

			r.Get("/absences", app.absenceHandler.ListAbsences)
			r.Post("/absences", app.absenceHandler.MarkAbsent)
			r.Delete("/absences/{date}", app.absenceHandler.RemoveAbsence)

			r.Get("/scores", app.scoreHandler.GetRanking)
			r.Get("/scores/justice", app.scoreHandler.GetJusticeChart)
			r.Get("/scores/{user_id}", app.scoreHandler.GetUserScore)
			r.Get("/badges/{user_id}", app.scoreHandler.GetUserBadges)

			r.With(handler.AdminOnly).Post("/admin/trigger-round", app.adminHandler.TriggerRound)
			r.With(handler.AdminOnly).Post("/admin/rounds", app.adminHandler.CreateRound)
			r.With(handler.AdminOnly).Post("/admin/rounds/{id}/force-close", app.adminHandler.ForceCloseRound)
			r.With(handler.AdminOnly).Put("/admin/rounds/{id}/payer", app.adminHandler.ChangePayer)
			r.With(handler.AdminOnly).Post("/admin/notifications/send", app.adminHandler.SendNotification)
			r.With(handler.AdminOnly).Put("/users/{id}/deactivate", app.userHandler.Deactivate)
			r.With(handler.AdminOnly).Put("/users/{id}/activate", app.userHandler.Activate)
		})
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
	config               config
	userHandler          *handler.UserHandler
	authHandler          *handler.AuthHandler
	configHandler        *handler.ConfigHandler
	rotationHandler      *handler.RotationHandler
	roundHandler         *handler.RoundHandler
	participationHandler *handler.ParticipationHandler
	deviceHandler        *handler.DeviceTokenHandler
	scoreHandler         *handler.ScoreHandler
	adminHandler         *handler.AdminHandler
	absenceHandler       *handler.AbsenceHandler
}

type config struct {
	addr      string
	db        dbConfig
	jwtSecret string
}

type dbConfig struct {
	dsn string
}
