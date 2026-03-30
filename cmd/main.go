package main

import (
	"log/slog"
	"os"

	"github.com/antoniobt12062002/pao-de-queijo/internal/db"
	handler "github.com/antoniobt12062002/pao-de-queijo/internal/handler/http"
	"github.com/antoniobt12062002/pao-de-queijo/internal/repository/postgres"
	"github.com/antoniobt12062002/pao-de-queijo/internal/usecase"
	"github.com/joho/godotenv"
)

// @title           API Pão de Queijo
// @version         1.0
// @description     API interna para o time de desenvolvimento.
// @host            localhost:8080
// @BasePath        /v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Setup logger first so all subsequent slog calls are structured
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found, using environment variables")
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		slog.Error("DB_DSN environment variable is required")
		os.Exit(1)
	}

	if err := db.RunMigrations(dsn); err != nil {
		slog.Error("failed to run migrations", "err", err)
		os.Exit(1)
	}

	gormDB, err := db.New(dsn)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}

	userRepo := postgres.NewUserRepository(gormDB)
	userUC := usecase.NewUserUseCase(userRepo, os.Getenv("JWT_SECRET"))
	userHandler := handler.NewUserHandler(userUC)
	authHandler := handler.NewAuthHandler(
		userUC,
		os.Getenv("GITHUB_CLIENT_ID"),
		os.Getenv("GITHUB_CLIENT_SECRET"),
		os.Getenv("GITHUB_CALLBACK_URL"),
	)
	configRepo    := postgres.NewConfigRepository(gormDB)
	configUC      := usecase.NewConfigUseCase(configRepo)
	configHandler := handler.NewConfigHandler(configUC)

	rotationRepo    := postgres.NewRotationRepository(gormDB)
	rotationUC      := usecase.NewRotationUseCase(rotationRepo, userRepo)
	rotationHandler := handler.NewRotationHandler(rotationUC)

	cfg := &config{
		addr:      ":8080",
		db:        dbConfig{dsn: dsn},
		jwtSecret: os.Getenv("JWT_SECRET"),
	}

	api := &application{
		config:          *cfg,
		userHandler:     userHandler,
		authHandler:     authHandler,
		configHandler:   configHandler,
		rotationHandler: rotationHandler,
	}

	if err := api.run(api.mount()); err != nil {
		slog.Error("error starting server", "err", err)
		os.Exit(1)
	}
}
