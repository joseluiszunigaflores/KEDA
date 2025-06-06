package main

import (
	"context"
	"log"
	"os"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

func main() {
	// KAFKA_BROKER apuntará a nuestro servicio de Kafka en Kubernetes
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	consumerGroup := os.Getenv("KAFKA_CONSUMER_GROUP")

	if kafkaBroker == "" || kafkaTopic == "" || consumerGroup == "" {
		log.Fatal("Environment variables KAFKA_BROKER, KAFKA_TOPIC, and KAFKA_CONSUMER_GROUP must be set.")
	}

	log.Printf("Starting Kafka consumer for topic '%s', group '%s' on broker '%s'", kafkaTopic, consumerGroup, kafkaBroker)

	// Configuración del lector de Kafka
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{kafkaBroker},
		Topic:       kafkaTopic,
		GroupID:     consumerGroup,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		MaxAttempts: 5,
		ErrorLogger: log.New(os.Stderr, "KAFKA ERROR: ", log.LstdFlags),
		Logger:      log.New(os.Stdout, "KAFKA INFO: ", log.LstdFlags),
	})
	defer reader.Close()

	ctx := context.Background()

	for {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("Error reading message: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf("Message received on partition %d, offset %d: %s", m.Partition, m.Offset, string(m.Value))
		// Simular carga de trabajo
		time.Sleep(5 * time.Second) // Simula un procesamiento que toma tiempo
		log.Printf("Finished processing message: %s", string(m.Value))
	}
}
