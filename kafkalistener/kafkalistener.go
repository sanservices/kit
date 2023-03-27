package kafkalistener

import (
	"context"
	"crypto/tls"
	"errors"
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

var ErrBrokerNotEnabled error = errors.New("message broker is not enabled")
var ErrPublishOnConsumeOnly error = errors.New("message broker was set as a consume only")

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
		ReconnectRetrySleep:   time.Second * 60,
	}

	routerConfig := message.RouterConfig{}
	router, err := message.NewRouter(routerConfig, watermillLogger)
	if err != nil {
		log.Println("Error creating router: ", err)
		return nil, err
	}

	return &MessageBroker{
		enabled:          true,
		subscriberConfig: subscriberConfig,
		publisher:        publisher,
		registryClient:   registryClient,
		logger:           watermillLogger,
		router:           router,
	}, nil
}

// SetSchema Gets the schema from the schema-registry,
//
// if it doesn't exists tries to register the schema.
func (mb *MessageBroker) SetSchema(topic *Topic) error {
	if !mb.enabled {
		return ErrBrokerNotEnabled
	}

	subject := topic.Name + "-value"
	var err error
	var schemaInfo registry.SchemaInfo
	var schemaID int

	// if the topic wont register the definition,
	// just grab the schema from the registry.
	if !topic.RegisterSchema {
		topic.Schema, err = mb.registryClient.GetLatestSchema(subject)
		return err
	}

	// return schema if it exists in the registry with the given avro definition.
	schemaID, topic.Schema, _ = mb.registryClient.IsRegistered(subject, topic.RawSchema)
	if schemaID > 0 {
		return nil
	}

	// Get the most recent schema from the registry.
	schemaInfo, err = mb.registryClient.GetLatestSchemaInfo(subject)
	if err == nil && (schemaInfo.Version >= 0 && schemaInfo.Version >= topic.Version) {
		topic.Schema = schemaInfo.Schema
		return nil
	}

	// Attempt to register the schema in the registry.
	_, topic.Schema, err = mb.registryClient.CreateSchema(subject, topic.RawSchema)
	return err
}

func (mb *MessageBroker) Publish(topic *Topic, data interface{}) error {
	if !mb.enabled {
		return ErrBrokerNotEnabled
	}

	if mb.publisher == nil {
		return ErrPublishOnConsumeOnly
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
	saramaConfig.Metadata.RefreshFrequency = time.Second * 30
	saramaConfig.Metadata.Timeout = time.Minute * 1

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
