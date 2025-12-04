package kafkalistener

import (
	"context"

	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
)

type RouteHandler struct {
	Name        string
	Topic       *Topic
	HandlerFunc message.NoPublishHandlerFunc
}

// SetRetry attempts to set the retry policy for the router.
func (mb *MessageBroker) SetRetry(retry *Retry) {
	if mb.router == nil {
		return
	}

	mb.router.AddMiddleware(retry.Middleware)
}

// Listen starts the router and the message broker. This call is blocking while the router is running.
//
// To stop Listen() you should call Stop().
func (mb *MessageBroker) Listen(ctx context.Context, handlers []RouteHandler) error {
	if !mb.enabled {
		return ErrBrokerNotEnabled
	}

	sub, err := kafka.NewSubscriber(mb.subscriberConfig, mb.logger)
	if err != nil {
		return err
	}

	for _, h := range handlers {
		err := mb.registerHandler(h, sub)
		if err != nil {
			return err
		}
	}

	mb.router.AddPlugin(plugin.SignalsHandler)
	mb.router.AddMiddleware(middleware.CorrelationID)

	return mb.router.Run(ctx)
}

// registerHandler sets the Schema and adds the handler to the router.
func (mb *MessageBroker) registerHandler(
	handler RouteHandler,
	subscriber *kafka.Subscriber,
) error {

	err := mb.SetSchema(handler.Topic)
	if err != nil {
		return err
	}

	mb.router.AddNoPublisherHandler(
		handler.Name,
		handler.Topic.Name,
		subscriber,
		handler.HandlerFunc,
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
