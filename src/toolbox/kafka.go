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
	reader *KafkaEnvironmentConfig
	writer *KafkaEnvironmentConfig
}

type MessageHandler func(string, string)

func (this *KafkaServer) StartNewEventReader(topic string, consumerGroup string, marshaller *EventMarshaller, handler EventHandler) {
	this.startNewReader(topic, consumerGroup, func(key string, value string) {
		id, t, data, err := marshaller.UnmarshallToType(value)
		if err != nil {
			fmt.Printf("Error processing eventId=%s, eventType=%s, err = %s", id, t, err)
		}
		handler(id, t, data)
	})
}

func (this *KafkaServer) startNewReader(topic string, consumerGroup string, handler MessageHandler) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{fmt.Sprintf("%s:%s", this.reader.Host, this.reader.Port)},
		GroupID:  consumerGroup,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

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
		log.Fatal("failed to close reader:", err)
	}
}

func (this *KafkaServer) StartNewWriter(topic string, marshaller *EventMarshaller) *EventWriter {
	address := fmt.Sprintf("%s:%s", this.writer.Host, this.writer.Port)
	w := &kafka.Writer{
		Addr:     kafka.TCP(address),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	_, err := kafka.DialLeader(context.Background(), "tcp", address, topic, 0)
	if err != nil {
		panic(err.Error())
	}

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

func LoadKafkaReaderEnvironmentConfig() *KafkaEnvironmentConfig {
	return &KafkaEnvironmentConfig{
		Host: os.Getenv("KAFKA_READER_HOST"),
		Port: os.Getenv("KAFKA_READER_PORT"),
	}
}

func LoadKafkaWriterEnvironmentConfig() *KafkaEnvironmentConfig {
	return &KafkaEnvironmentConfig{
		Host: os.Getenv("KAFKA_WRITER_HOST"),
		Port: os.Getenv("KAFKA_WRITER_PORT"),
	}
}

func InitKafkaDefault() *KafkaServer {
	return InitKafka(LoadKafkaReaderEnvironmentConfig(), LoadKafkaWriterEnvironmentConfig())
}

func InitKafka(reader *KafkaEnvironmentConfig, writer *KafkaEnvironmentConfig) *KafkaServer {
	return &KafkaServer{reader: reader, writer: writer}
}