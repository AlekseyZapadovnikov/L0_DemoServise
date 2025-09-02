package broker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Asus/L0_DemoServise/internal/entity"
	"github.com/segmentio/kafka-go"
	"log/slog"
)

type OrderSaver interface {
	SaveOrder(ctx context.Context, o entity.Order) error // Интерфейс для вызова из service
}

type KafkaConsumer struct {
	reader *kafka.Reader
	saver  OrderSaver
}

func NewKafkaConsumer(brokerAddr string, topic string, groupID string, saver OrderSaver) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddr}, // e.g. "localhost:9092" брокеры, которые подключены к кластеру
		Topic:    topic,                // "orders"
		GroupID:  groupID,              // "order-service-group"
		MaxBytes: 10e6,                 // 10MB
	})
	return &KafkaConsumer{reader: reader, saver: saver}
}

// запускаем в отдельной горутине, этот метод принимает сообщения из Kafka и сохраняет сообщения в БД и Cache
func (c *KafkaConsumer) ConsumeAndSave(ctx context.Context) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return fmt.Errorf("failed to read message: %w", err)
		}

		var order entity.Order
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			slog.Error("failed to parse order JSON", "error", err)
			continue // Пропускаем некорректное сообщение, предварительно логируя
		}

		// Сохраняем в БД и кэш через saver (service)
		if err := c.saver.SaveOrder(ctx, order); err != nil {
			slog.Error("failed to save order", "order_uid", order.OrderUID, "error", err)
			continue
		}

		slog.Info("Order processed from Kafka", "order_uid", order.OrderUID)
	}
}
// немного не SingleResp, но мне кажется удобно, просто в горутине запустить и уже с этим не возиться
// работать только с Cache

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}
