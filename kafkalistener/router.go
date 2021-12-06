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

func StartListener(ctx context.Context, mb *MessageBroker, handlers []RouteHandler) error {
	if !mb.enabled {
		return nil
	}

	return mb.Listen(ctx, handlers)
}

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

	go router.Run(ctx)
	return nil
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
