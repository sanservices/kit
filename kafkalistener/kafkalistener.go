package kafkalistener

import (
	"context"
	"crypto/tls"
	"log"
	"os"
	"time"

	"github.com/Shopify/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v2/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/hamba/avro"
	"github.com/hamba/avro/registry"
	tlskit "github.com/sanservices/kit/tls"
)

type KafkaConfig struct {
	Enabled         bool       `yaml:"enabled"`
	Version         string     `yaml:"version"`
	ConsumeOnly     bool       `yaml:"consume_only"`
	ConsumerGroupID string     `yaml:"consumer_group_id"`
	FromOldest      bool       `yaml:"from_oldest"`
	SchemaReg       string     `yaml:"schema_registration"`
	Brokers         []string   `yaml:"brokers"`
	TLS             tlskit.TLS `yaml:"TLS"`
}

type MessageBroker struct {
	enabled          bool
	publisher        *kafka.Publisher
	subscriberConfig kafka.SubscriberConfig
	registryClient   *registry.Client
	logger           watermill.LoggerAdapter
	router           *message.Router
	canCreateSchema  bool
}

type Topic struct {
	Name      string
	Version   int
	RawSchema string
	Schema    avro.Schema
}

func New(
	ctx context.Context,
	config *KafkaConfig,
	debug bool,
) (*MessageBroker, error) {

	log.Println("Creating message broker...")
	if !config.Enabled {
		return &MessageBroker{enabled: false}, nil
	}

	var publisher *kafka.Publisher

	tlsConfig, err := tlskit.GetTLSConf(config.TLS)
	if err != nil {
		log.Println("Error getting TLS configuration: ", err)
		return nil, err
	}

	saramaConfig := setSaramaConfig(tlsConfig)
	watermillLogger := watermill.NewStdLoggerWithOut(os.Stdout, debug, debug)

	publisher, err = configurePublisher(config, saramaConfig, watermillLogger)
	if err != nil {
		log.Println("Error creating publisher: ", err)
		return nil, err
	}

	registryClient, err := GetRegistryClient(tlsConfig, config.SchemaReg)
	if err != nil {
		log.Println("Error creating registry client: ", err)
		return nil, err
	}

	subscriberConfig := kafka.SubscriberConfig{
		Brokers:               config.Brokers,
		Unmarshaler:           kafka.DefaultMarshaler{},
		OverwriteSaramaConfig: saramaConfig,
		ConsumerGroup:         config.ConsumerGroupID,
		NackResendSleep:       time.Second * 60,
	}

	return &MessageBroker{
		enabled:          true,
		subscriberConfig: subscriberConfig,
		publisher:        publisher,
		registryClient:   registryClient,
		logger:           watermillLogger,
	}, nil
}

// SetSchema Gets the schema from the schema-registry,
//
// if it doesn't exists tries to register the schema.
func (mb *MessageBroker) SetSchema(topic *Topic) error {

	subject := topic.Name + "-value"
	var err error
	var schema avro.Schema
	var schemaInfo registry.SchemaInfo
	var schemaID int

	if mb.canCreateSchema {
		schemaID, schema, _ = mb.registryClient.IsRegistered(subject, topic.RawSchema)
		if schemaID > 0 {
			topic.Schema = schema
			return nil
		}

		schemaInfo, err = mb.registryClient.GetLatestSchemaInfo(subject)
		if err != nil || schemaInfo.Version < topic.Version {
			_, schema, err = mb.registryClient.CreateSchema(subject, topic.RawSchema)
			if err != nil {
				log.Println("Error creating schema: ", err)
				return err
			}

		} else {
			schema = schemaInfo.Schema
		}

	} else {
		schema, err = mb.registryClient.GetLatestSchema(subject)
		if err != nil {
			log.Println("Error getting schema from registry: ", err)
			return err
		}
	}

	topic.Schema = schema
	return nil
}

func (mb *MessageBroker) Publish(topic *Topic, data interface{}) error {
	if mb.publisher == nil {
		return nil
	}

	messageToSend, err := avro.Marshal(topic.Schema, data)
	if err != nil {
		return err
	}

	msg := message.NewMessage(watermill.NewUUID(), messageToSend)
	return mb.publisher.Publish(topic.Name, msg)
}

func GetRegistryClient(tlsConfig *tls.Config, schemaReg string) (*registry.Client, error) {
	httpsClient := tlskit.GetHTTPSClient(tlsConfig)

	return registry.NewClient(schemaReg, registry.WithHTTPClient(httpsClient))
}

func setSaramaConfig(tlsConfig *tls.Config) *sarama.Config {
	saramaConfig := kafka.DefaultSaramaSubscriberConfig()

	saramaConfig.Net.TLS.Config = tlsConfig
	saramaConfig.Net.TLS.Enable = true
	saramaConfig.Version = sarama.V2_5_0_0
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	// Producer tweaks
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Retry.Backoff = time.Second * 30
	saramaConfig.Producer.Retry.Max = 50 // A very high number to ensure the message is written (infinity could be better)
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Net.MaxOpenRequests = 1
	saramaConfig.Producer.Idempotent = true

	return saramaConfig
}

func configurePublisher(
	config *KafkaConfig,
	saramaConfig *sarama.Config,
	logger watermill.LoggerAdapter,
) (*kafka.Publisher, error) {
	if config.ConsumeOnly {
		return nil, nil
	}

	publisherConfig := kafka.PublisherConfig{
		Brokers:               config.Brokers,
		Marshaler:             kafka.DefaultMarshaler{},
		OverwriteSaramaConfig: saramaConfig,
	}

	return kafka.NewPublisher(publisherConfig, logger)
}
