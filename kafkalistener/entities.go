package kafkalistener

import (
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/hamba/avro"
	"github.com/hamba/avro/registry"
	"github.com/sanservices/kit/tls"
)

type KafkaConfig struct {
	Enabled         bool     `yaml:"enabled"`
	Version         string   `yaml:"version"`
	ConsumeOnly     bool     `yaml:"consume_only"`
	ConsumerGroupID string   `yaml:"consumer_group_id"`
	FromOldest      bool     `yaml:"from_oldest"`
	SchemaReg       string   `yaml:"schema_registration"`
	Brokers         []string `yaml:"brokers"`
	TLS             tls.TLS  `yaml:"TLS"`
	// GroupInstanceID enables Kafka static group membership (KIP-345).
	// Must be stable across restarts and unique per replica (e.g. K8s StatefulSet pod name).
	// When set, the broker skips rebalancing if the pod restarts within SessionTimeout.
	// Leave empty to disable.
	GroupInstanceID string `yaml:"group_instance_id"`
	// SessionTimeout is how long the broker waits for a heartbeat before declaring the
	// consumer dead and triggering a rebalance. Defaults to 45s when unset.
	// With static membership, this doubles as the OOMKill-restart window — set it to
	// comfortably exceed your worst-case pod restart time.
	// Heartbeat interval is automatically set to SessionTimeout/3.
	SessionTimeout time.Duration `yaml:"session_timeout"`
}

type MessageBroker struct {
	enabled          bool
	publisher        *kafka.Publisher
	subscriberConfig kafka.SubscriberConfig
	registryClient   *registry.Client
	logger           watermill.LoggerAdapter
	router           *message.Router
}

type Topic struct {
	// Name is the name of the topic.
	Name string
	// Version is the version of the topica in the schema registry.
	Version int
	// Schema is the avro definition of the topic.
	RawSchema string
	// Schema is avro schema from the schema registry.
	Schema avro.Schema
	// RegisterSchema indicates if the rawSchema should be registered
	// in the kafka's schema registry.
	RegisterSchema bool
}
