# Overview
This package provides the boilerplate to create a connection to the kafka server
and setup the configurations of the kafka client. This package was built on top of the [watermill framework](https://github.com/ThreeDotsLabs/watermill).

## Usage

1. First install the kit package
```go
go get github.com/sanservices/kit
```

2. Then import the library
```go
import github.com/sanservices/kit/kafkalistener
```

3. After that you can use the functions
```go

// TestPayload is the DTO for incoming messages.
type TestPayload struct {
	Name string
	Age  int
}

// Define the topic to be used
var topicTest = &kafkalistener.Topic{
	Name: "test",
}

func main() {
	ctx := context.Background()

    // Set the kafka configurations
	config := &kafkalistener.KafkaConfig{
		Enabled:         true,
		Brokers:         []string{"localhost:9092"},
		Version:         "2.1.1",
		ConsumeOnly:     true,
		ConsumerGroupID: "consumer-test",
		FromOldest:      true,
		SchemaReg:       "http://localhost:8081",
        // The kafkalistener package only works with TLS connections
		TLS: tls.TLS{
			CACertPEM:  "./cacert.pem",
			CertPEM:    "./cert.pem",
			KeyPEM:     "./key.pem",
			SkipVerify: true,
		},
	}

    // Initialize the message broker
	messageBroker, err := kafkalistener.New(ctx, config, true)
	if err != nil {
		log.Fatal(err)
	}

    // Set the handler for each topic
	routeHandlers := []kafkalistener.RouteHandler{
		{
			Topic:       topicTest,
			HandlerFunc: handlerForTest,
		},
	}

    // Start listening to kafka
	err = kafkalistener.StartListener(ctx, messageBroker, routeHandlers)
	if err != nil {
		log.Fatal(err)
	}
}

// handlerForTest Handles incoming messages for topic "test"
func handlerForTest(msg *message.Message) error {
	data := TestPayload{}

	err := kafkalistener.DecodePayload(topicTest, msg.Payload, &data)
	if err != nil {
		return err
	}

	log.Printf("Received message: %+v", data)
	return nil
}
```