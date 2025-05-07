package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"banking-service/configs"
	"banking-service/internal/handler"
	"banking-service/internal/middleware"
	"banking-service/internal/repository"
	"banking-service/internal/service"
	"banking-service/pkg/scheduler"
)

func main() {
	// Initialize logger
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)

	// Load configuration
	cfg, err := configs.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	db, err := initDB(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	repos := repository.NewRepository(db)

	// Initialize services
	services := service.NewService(service.Dependencies{
		Repos:       repos,
		Logger:      log,
		Config:      cfg,
	})

	// Initialize handlers
	handlers := handler.NewHandler(handler.Dependencies{
		Services:    services,
		Logger:      log,
		Config:      cfg,
	})

	// Initialize router
	router := mux.NewRouter()
	
	// Public routes
	router.HandleFunc("/register", handlers.User.Register).Methods(http.MethodPost)
	router.HandleFunc("/login", handlers.User.Login).Methods(http.MethodPost)

	// Protected routes with middleware
	api := router.PathPrefix("/api").Subrouter()
	api.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	api.Use(middleware.LogMiddleware(log))

	// Account endpoints
	api.HandleFunc("/accounts", handlers.Account.Create).Methods(http.MethodPost)
	api.HandleFunc("/accounts", handlers.Account.GetAll).Methods(http.MethodGet)
	api.HandleFunc("/accounts/{id}", handlers.Account.GetByID).Methods(http.MethodGet)
	api.HandleFunc("/accounts/{id}/balance", handlers.Account.UpdateBalance).Methods(http.MethodPut)
	api.HandleFunc("/accounts/{id}/predict", handlers.Analytics.PredictBalance).Methods(http.MethodGet)

	// Card endpoints
	api.HandleFunc("/cards", handlers.Card.Create).Methods(http.MethodPost)
	api.HandleFunc("/cards", handlers.Card.GetAll).Methods(http.MethodGet)
	api.HandleFunc("/cards/{id}", handlers.Card.GetByID).Methods(http.MethodGet)

	// Transaction endpoints
	api.HandleFunc("/transfer", handlers.Transaction.Transfer).Methods(http.MethodPost)
	api.HandleFunc("/transactions", handlers.Transaction.GetAll).Methods(http.MethodGet)

	// Credit endpoints
	api.HandleFunc("/credits", handlers.Credit.Create).Methods(http.MethodPost)
	api.HandleFunc("/credits", handlers.Credit.GetAll).Methods(http.MethodGet)
	api.HandleFunc("/credits/{id}", handlers.Credit.GetByID).Methods(http.MethodGet)
	api.HandleFunc("/credits/{id}/schedule", handlers.Credit.GetSchedule).Methods(http.MethodGet)

	// Analytics endpoints
	api.HandleFunc("/analytics", handlers.Analytics.GetStatistics).Methods(http.MethodGet)

	// Start the payment scheduler
	paymentScheduler := scheduler.NewScheduler(services.Credit, log)
	paymentScheduler.Start(time.Hour * 24) // Check payments once per day
	defer paymentScheduler.Stop()

	// Configure and start server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Second * 15,
		WriteTimeout: time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	// Start the server in a goroutine
	go func() {
		log.Infof("Starting server on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	// Create a deadline context for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Shutdown the server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Info("Server gracefully stopped")
}

func initDB(cfg *configs.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.DBName)
	
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}