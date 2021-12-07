package kafkalistener

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill-kafka/v2/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
)

type RouteHandler struct {
	Name        string
	Topic       *Topic
	HandlerFunc message.NoPublishHandlerFunc
}

// Listen starts the router and the message broker. This call is blocking while the router is running.
//
// To stop Listen() you should call Stop().
func (mb *MessageBroker) Listen(ctx context.Context, handlers []RouteHandler) error {
	config := message.RouterConfig{}
	router, err := message.NewRouter(config, mb.logger)
	if err != nil {
		return err
	}

	subscriber, err := kafka.NewSubscriber(mb.subscriberConfig, mb.logger)
	if err != nil {
		return err
	}

	for _, r := range handlers {
		err := mb.registerHandler(router, subscriber, r.Name, r.Topic, r.HandlerFunc)
		if err != nil {
			return err
		}
	}

	retryMidd := Retry{
		MaxRetries:         5,
		InitialInterval:    time.Millisecond * 500,
		Multiplier:         2.5,
		MaxInterval:        time.Second * 5,
		MaxElapsedTime:     time.Minute * 2,
		AckAfterMaxRetries: true,
		Logger:             mb.logger,
	}.Middleware

	router.AddPlugin(plugin.SignalsHandler)
	router.AddMiddleware(middleware.CorrelationID)
	router.AddMiddleware(retryMidd)
	mb.router = router

	return mb.router.Run(ctx)
}

// registerHandler sets the Schema and adds the handler to the router.
func (mb *MessageBroker) registerHandler(
	router *message.Router,
	subscriber *kafka.Subscriber,
	name string,
	topic *Topic,
	handlerFunc message.NoPublishHandlerFunc,
) error {

	err := mb.SetSchema(topic)
	if err != nil {
		return err
	}

	router.AddNoPublisherHandler(
		name,
		topic.Name,
		subscriber,
		handlerFunc,
	)

	return nil
}

// Stop gracefully closes the router with a timeout provided in the configuration.
func (mb *MessageBroker) Stop() error {
	if mb.router != nil {
		return mb.router.Close()
	}

	return nil
}
