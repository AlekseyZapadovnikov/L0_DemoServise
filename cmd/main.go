package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/Asus/L0_DemoServise/config"
	"github.com/Asus/L0_DemoServise/internal/broker"
	"github.com/Asus/L0_DemoServise/internal/entity"
	"github.com/Asus/L0_DemoServise/internal/server"
	"github.com/Asus/L0_DemoServise/internal/service"
	"github.com/Asus/L0_DemoServise/internal/storage"
)

func main() {
	cfg := config.MustLoad()
	slog.Info("Configuration loaded successfully")

	stor, err := storage.NewStorage(&cfg.Storage)
	if err != nil {
		slog.Error("failed to init storage", "error", err)
		os.Exit(1)
	}
	slog.Info("Successfully connected to DB", "host", cfg.Storage.Host, "port", cfg.Storage.Port, "dbname", cfg.Storage.DBName)
	defer stor.Close()

	service := &service.Service{
		OrderMap:   make(map[string]entity.Order),
		OrderTaker: stor, // Storage реализует OrderRepository
	}
	slog.Info("Service layer initialized")

	// Восстановление кэша
	if err := service.LoadCache(context.Background()); err != nil {
		slog.Error("failed to load cache", "error", err)
		os.Exit(1)
	}
	slog.Info("Cache successfully populated from database", "orders_loaded", len(service.OrderMap))

	// Kafka consumer
	consumer := broker.NewKafkaConsumer("localhost:9092", "orders", "order-service-group", service)
	slog.Info("Kafka consumer initialized")
	go func() {
		if err := consumer.ConsumeAndSave(context.Background()); err != nil {
			slog.Error("consumer error", "error", err)
		}
	}()

	server := server.NewServer("localhost:8080", service)
	slog.Info("HTTP server initialized", "address", "localhost:8080")
	server.Start()
}