package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

func main() {
	// KAFKA_BROKER apuntará a nuestro servicio de Kafka en Kubernetes
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	messageCountStr := os.Getenv("MESSAGE_COUNT")
	sendIntervalMsStr := os.Getenv("SEND_INTERVAL_MS")

	if kafkaBroker == "" || kafkaTopic == "" {
		log.Fatal("Environment variables KAFKA_BROKER and KAFKA_TOPIC must be set.")
	}

	messageCount := 1
	if messageCountStr != "" {
		parsedCount, err := strconv.Atoi(messageCountStr)
		if err == nil {
			messageCount = parsedCount
		}
	}

	sendIntervalMs := 100 // Default to 100ms
	if sendIntervalMsStr != "" {
		parsedInterval, err := strconv.Atoi(sendIntervalMsStr)
		if err == nil {
			sendIntervalMs = parsedInterval
		}
	}

	// Configuración del escritor de Kafka
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{kafkaBroker},
		Topic:   kafkaTopic,
		Balancer: &kafka.LeastBytes{}, // Balanceador para distribuir mensajes entre particiones
	})
	defer writer.Close()

	log.Printf("Starting Kafka producer for topic '%s' on broker '%s'", kafkaTopic, kafkaBroker)

	for i := 0; i < messageCount; i++ {
		msgValue := fmt.Sprintf("Hello Kafka! Message %d from Producer at %s", i+1, time.Now().Format("15:04:05"))
		err := writer.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(fmt.Sprintf("Key-%d", i)),
				Value: []byte(msgValue),
			},
		)
		if err != nil {
			log.Printf("Failed to write message %d: %v", i+1, err)
		} else {
			log.Printf("Sent message: %s", msgValue)
		}
		time.Sleep(time.Duration(sendIntervalMs) * time.Millisecond) // Intervalo entre mensajes
	}

	log.Printf("Finished sending %d messages to topic '%s'.", messageCount, kafkaTopic)
}