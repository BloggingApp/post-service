package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BloggingApp/post-service/internal/config"
	"github.com/BloggingApp/post-service/internal/handler"
	"github.com/BloggingApp/post-service/internal/rabbitmq"
	"github.com/BloggingApp/post-service/internal/repository"
	"github.com/BloggingApp/post-service/internal/repository/postgres"
	"github.com/BloggingApp/post-service/internal/server"
	"github.com/BloggingApp/post-service/internal/service"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	logger, _ := zap.NewProduction()

	if err := loadEnv(); err != nil {
		logger.Sugar().Panicf("failed to load environment variables: %s", err.Error())
	}

	if err := initConfig(); err != nil {
		logger.Sugar().Panicf("failed to initialize yaml config: %s", err.Error())
	}

	dbConfig := config.DBConfig{
		Username: os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		Host: os.Getenv("POSTGRES_HOST"),
		Port: os.Getenv("POSTGRES_PORT"),
		DBName: os.Getenv("POSTGRES_DATABASE"),
		SSLMode: os.Getenv("POSTGRES_SSLMODE"),
	}
	db, err := postgres.DB(ctx, dbConfig)
	if err != nil {
		logger.Sugar().Panicf("failed to connect to postgres: %s", err.Error())
	}
	if err := db.Ping(ctx); err != nil {
		logger.Sugar().Panicf("failed to ping postgres: %s", err.Error())
	}
	logger.Info("Successfully connected to PostgreSQL")

	redisOptions := &redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	}
	rdb := redis.NewClient(redisOptions)
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		logger.Sugar().Panicf("failed to ping redis: %s", err.Error())
	}
	logger.Sugar().Infof("Successfully connected to Redis: %s", pong)

	rabbitmq, err := rabbitmq.New(os.Getenv("RABBITMQ_CONN_STRING"))
	if err != nil {
		logger.Sugar().Panicf("failed to connect to rabbitmq: %s", err.Error())
	}
	logger.Info("Successfully connected to RabbitMQ")

	repos := repository.New(db, rdb, logger)
	services := service.New(logger, repos, rabbitmq)
	handlers := handler.New(services)

	srv := server.New()
	serverConfig := config.ServerConfig{
		Port: viper.GetString("app.port"),
		Handler: handlers.InitRoutes(),
		MaxHeaderBytes: 1 << 20,
		ReadTimeout: time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	go func(srv server.Server, cfg config.ServerConfig) {
		if err := srv.Run(cfg); err != nil {
			logger.Sugar().Panicf("failed to run http server: %s", err.Error())
		}
	}(*srv, serverConfig)

	go services.StartConsumeAll(ctx)

	logger.Info("Server started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logger.Info("Server shutting down")
}

func loadEnv() error {
	return godotenv.Load()
}

func initConfig() error {
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.SetConfigName("app")
	return viper.ReadInConfig()
}
