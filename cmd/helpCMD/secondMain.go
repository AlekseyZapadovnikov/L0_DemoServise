package main

import (
	"context"
	"fmt"
	"os"

	"github.com/segmentio/kafka-go"
)

func main() {
	writer := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "orders",
		Balancer: &kafka.LeastBytes{},
	}

	jsonData, err := os.ReadFile("cmd/helpCMD/model.json") // Твой model.json в корне
	if err != nil {
		fmt.Println("failed to read json", err)
		return
	}

	err = writer.WriteMessages(context.Background(), kafka.Message{Value: jsonData})
	if err != nil {
		fmt.Println("failed to write", err)
	}
	writer.Close()
	fmt.Println("Message sent")
}

// это мини "скрипт" для отправки сообщения в kafka