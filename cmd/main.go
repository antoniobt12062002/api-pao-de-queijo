package main

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/robfig/cron/v3"
	"github.com/antoniobt12062002/pao-de-queijo/internal/db"
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	handler "github.com/antoniobt12062002/pao-de-queijo/internal/handler/http"
	"github.com/antoniobt12062002/pao-de-queijo/internal/job"
	"github.com/antoniobt12062002/pao-de-queijo/internal/notification"
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

	firebaseCredentials := os.Getenv("FIREBASE_CREDENTIALS_JSON")

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

	roundRepo    := postgres.NewRoundRepository(gormDB)
	noopScore    := &domain.NoopScoreUpdater{}

	participationRepo    := postgres.NewParticipationRepository(gormDB)
	participationUC      := usecase.NewParticipationUseCase(participationRepo, roundRepo)
	participationHandler := handler.NewParticipationHandler(participationUC)

	deviceTokenRepo := postgres.NewDeviceTokenRepository(gormDB)
	notifLogRepo    := postgres.NewNotificationLogRepository(gormDB)

	var notifySvc domain.NotificationService
	if firebaseCredentials != "" {
		fcmSvc, err := notification.NewFCMNotificationService([]byte(firebaseCredentials), deviceTokenRepo, notifLogRepo)
		if err != nil {
			slog.Error("failed to initialize Firebase FCM", "err", err)
			os.Exit(1)
		}
		notifySvc = fcmSvc
		slog.Info("Firebase FCM notification service initialized")
	} else {
		slog.Warn("FIREBASE_CREDENTIALS_JSON not set — using noop notification service")
		notifySvc = &domain.NoopNotificationService{}
	}

	roundUC      := usecase.NewRoundUseCase(roundRepo, rotationRepo, notifySvc)
	roundHandler := handler.NewRoundHandler(roundUC)

	deviceTokenUC      := usecase.NewDeviceTokenUseCase(deviceTokenRepo)
	deviceTokenHandler := handler.NewDeviceTokenHandler(deviceTokenUC)

	// Background jobs
	closer  := job.NewParticipationWindowCloser(roundRepo, notifySvc, noopScore)
	reminder := job.NewReminderSender(roundRepo, participationRepo, notifySvc)
	creator  := job.NewDailyRoundCreator(roundRepo, rotationRepo, configRepo, notifySvc, closer, reminder)

	// Scheduler: lê notify_at da config para montar expressão cron
	notifyAt := "08:00"
	if configs, err := configRepo.GetAll(); err == nil {
		for _, c := range configs {
			if c.Key == "notify_at" {
				notifyAt = c.Value
			}
		}
	}
	parts := strings.Split(notifyAt, ":")
	hour, minute := "8", "0"
	if len(parts) == 2 {
		hour = strconv.Itoa(mustAtoi(parts[0]))
		minute = strconv.Itoa(mustAtoi(parts[1]))
	}
	cronExpr := "0 " + minute + " " + hour + " * * *"

	c := cron.New(cron.WithSeconds())
	if _, err := c.AddFunc(cronExpr, creator.Run); err != nil {
		slog.Error("failed to schedule DailyRoundCreator", "err", err)
		os.Exit(1)
	}
	c.Start()
	slog.Info("cron scheduler started", "DailyRoundCreator", cronExpr)

	cfg := &config{
		addr:      ":8080",
		db:        dbConfig{dsn: dsn},
		jwtSecret: os.Getenv("JWT_SECRET"),
	}

	api := &application{
		config:               *cfg,
		userHandler:          userHandler,
		authHandler:          authHandler,
		configHandler:        configHandler,
		rotationHandler:      rotationHandler,
		roundHandler:         roundHandler,
		participationHandler: participationHandler,
		deviceHandler:        deviceTokenHandler,
	}

	if err := api.run(api.mount()); err != nil {
		slog.Error("error starting server", "err", err)
		os.Exit(1)
	}
}

func mustAtoi(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
