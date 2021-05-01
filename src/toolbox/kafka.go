package toolbox

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log"
	"os"
)

type KafkaEnvironmentConfig struct {
	Host string
	Port string
}

type KafkaServer struct {
	broker *KafkaEnvironmentConfig
}

type MessageHandler func(string, string)

func (this *KafkaServer) StartNewEventReader(topic string, consumerGroup string, marshaller *EventMarshaller, handler EventHandler) {
	go this.startNewReader(topic, consumerGroup, func(key string, value string) {
		id, t, data, err := marshaller.UnmarshallToType(value)
		if err != nil {
			fmt.Printf("Error processing eventId=%s, eventType=%s, err = %s", id, t, err)
		}
		handler(id, t, data)
	})
}

func (this *KafkaServer) startNewReader(topic string, consumerGroup string, handler MessageHandler) {
	address := fmt.Sprintf("%s:%s", this.broker.Host, this.broker.Port)
	log.Printf("Starting kafka broker (address %s)(topic %s)(consumer group %s)", address, topic, consumerGroup)
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{address},
		GroupID:  consumerGroup,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	log.Printf("Kafka broker (address %s)(topic %s)(consumer group %s) started", address, topic, consumerGroup)

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			break
		}
		key, value := string(m.Key), string(m.Value)
		fmt.Printf("message at offset %d: %s = %s\n", m.Offset, key, value)
		handler(key, value)
	}

	if err := r.Close(); err != nil {
		log.Fatal("failed to close broker:", err)
	}
}

func (this *KafkaServer) StartNewWriter(topic string, marshaller *EventMarshaller) *EventWriter {
	address := fmt.Sprintf("%s:%s", this.broker.Host, this.broker.Port)
	log.Printf("Starting kafka writer (address %s)(topic %s)", address, topic)
	w := &kafka.Writer{
		Addr:     kafka.TCP(address),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	_, err := kafka.DialLeader(context.Background(), "tcp", address, topic, 0)
	if err != nil {
		panic(err.Error())
	} else {
		log.Printf("Kafka force created topic %s", topic)
	}

	log.Printf("Kafka writer created (topic %s)", topic)

	return &EventWriter{
		writer:     w,
		marshaller: marshaller,
	}
}

type EventWriter struct {
	writer     *kafka.Writer
	marshaller *EventMarshaller
}

func (this *EventWriter) WriteEvent(eventType string, data interface{}) (string, error) {
	id, datastr, err := this.marshaller.Marshall(eventType, data)
	if err != nil {
		log.Println("failed to marshall messages:", err)
		return "", err
	} else {
		err = this.write(id, datastr)
		return id, err
	}
}

func (this *EventWriter) write(key string, value string) error {
	err := this.writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(key),
			Value: []byte(value),
		},
	)
	if err != nil {
		log.Println("failed to write messages:", err)
	}
	return err
}

func LoadKafkaEnvironmentConfig() *KafkaEnvironmentConfig {
	return &KafkaEnvironmentConfig{
		Host: os.Getenv("KAFKA_BROKER_HOST"),
		Port: os.Getenv("KAFKA_BROKER_PORT"),
	}
}

func InitKafkaDefault() *KafkaServer {
	return InitKafka(LoadKafkaEnvironmentConfig())
}

func InitKafka(broker *KafkaEnvironmentConfig) *KafkaServer {
	return &KafkaServer{broker: broker}
}
