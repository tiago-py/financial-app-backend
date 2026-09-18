package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"financial-app-backend/internal/auth"
	"financial-app-backend/internal/cache"
	"financial-app-backend/internal/config"
	"financial-app-backend/internal/controller"
	"financial-app-backend/internal/database"
	"financial-app-backend/internal/repository"
	"financial-app-backend/internal/router"
	"financial-app-backend/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuracao invalida", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("postgres indisponivel", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisCache, err := cache.Open(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		logger.Error("redis indisponivel", "error", err)
		os.Exit(1)
	}
	defer func() { _ = redisCache.Close() }()

	store := repository.New(pool)
	tokenManager := auth.NewManager(cfg.JWTSecret, cfg.JWTTTL)
	appService := service.New(store, redisCache, tokenManager)
	if err := appService.BootstrapAdmin(ctx, cfg.AdminName, cfg.AdminEmail, cfg.AdminPassword); err != nil {
		logger.Error("nao foi possivel preparar o administrador", "error", err)
		os.Exit(1)
	}
	handler := controller.New(appService)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router.New(handler, appService, cfg.CORSOrigin, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("api iniciada", "address", cfg.HTTPAddr, "environment", cfg.Environment)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("servidor encerrado", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("falha no encerramento", "error", err)
	}
}
