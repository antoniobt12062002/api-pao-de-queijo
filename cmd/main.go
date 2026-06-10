package main

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // embed IANA timezone DB so time.LoadLocation works on Railway

	"github.com/robfig/cron/v3"
	"github.com/antoniobt12062002/pao-de-queijo/internal/db"
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	handler "github.com/antoniobt12062002/pao-de-queijo/internal/handler/http"
	"github.com/antoniobt12062002/pao-de-queijo/internal/job"
	"github.com/antoniobt12062002/pao-de-queijo/internal/notification"
	"github.com/antoniobt12062002/pao-de-queijo/internal/repository/postgres"
	"github.com/antoniobt12062002/pao-de-queijo/internal/score"
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

	// Set server timezone so all time.Now() calls use Brazil time.
	// Railway runs in UTC; override here so date comparisons are correct.
	tzName := os.Getenv("TZ_LOCATION")
	if tzName == "" {
		tzName = "America/Sao_Paulo"
	}
	if loc, err := time.LoadLocation(tzName); err == nil {
		time.Local = loc
		slog.Info("timezone configured", "location", tzName)
	} else {
		slog.Warn("could not load timezone, defaulting to UTC", "err", err)
	}

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

	roundRepo := postgres.NewRoundRepository(gormDB)

	participationRepo    := postgres.NewParticipationRepository(gormDB)
	participationUC      := usecase.NewParticipationUseCase(participationRepo, roundRepo, userRepo)
	participationHandler := handler.NewParticipationHandler(participationUC)

	deviceTokenRepo := postgres.NewDeviceTokenRepository(gormDB)
	notifLogRepo    := postgres.NewNotificationLogRepository(gormDB)
	scoreRepo       := postgres.NewScoreRepository(gormDB)
	badgeRepo       := postgres.NewBadgeRepository(gormDB)

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

	scoreUpdater  := score.NewScoreUpdater(roundRepo, participationRepo, configRepo, scoreRepo)
	badgeChecker  := score.NewBadgeChecker(roundRepo, participationRepo, configRepo, scoreRepo, badgeRepo)

	roundUC      := usecase.NewRoundUseCase(roundRepo, rotationRepo, notifySvc, scoreUpdater)
	roundHandler := handler.NewRoundHandler(roundUC)

	deviceTokenUC      := usecase.NewDeviceTokenUseCase(deviceTokenRepo)
	deviceTokenHandler := handler.NewDeviceTokenHandler(deviceTokenUC)

	scoreUC      := usecase.NewScoreUseCase(scoreRepo, badgeRepo, userRepo)
	scoreHandler := handler.NewScoreHandler(scoreUC)

	// Background jobs
	closer  := job.NewParticipationWindowCloser(roundRepo, notifySvc, scoreUpdater, badgeChecker)
	reminder := job.NewReminderSender(roundRepo, participationRepo, notifySvc)
	creator  := job.NewDailyRoundCreator(roundRepo, rotationRepo, configRepo, notifySvc, closer, reminder)

	adminHandler := handler.NewAdminHandler(creator, closer, roundUC)

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

	c := cron.New(cron.WithSeconds(), cron.WithLocation(time.Local))
	if _, err := c.AddFunc(cronExpr, creator.Run); err != nil {
		slog.Error("failed to schedule DailyRoundCreator", "err", err)
		os.Exit(1)
	}
	c.Start()
	slog.Info("cron scheduler started", "DailyRoundCreator", cronExpr)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	cfg := &config{
		addr:      ":" + port,
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
		scoreHandler:         scoreHandler,
		adminHandler:         adminHandler,
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
