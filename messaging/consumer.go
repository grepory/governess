package messaging

import (
	"fmt"
	"net/url"
	"time"

	"github.com/bitly/go-nsq"
)

type Consumer struct {
	Topic      string
	RoutingKey string

	channel     chan EventInterface
	nsqConsumer *nsq.Consumer
	nsqConfig   *nsq.Config
}

// NewConsumer will create a named channel on the specified topic and return
// the associated message-producing channel.
func NewConsumer(topicName string, routingKey string) (*Consumer, error) {
	channel := make(chan EventInterface, 1)

	consumer := &Consumer{
		Topic:      topicName,
		RoutingKey: routingKey,
		nsqConfig:  nsq.NewConfig(),
		channel:    channel,
	}

	nsqConsumer, err := nsq.NewConsumer(topicName, routingKey, consumer.nsqConfig)
	if err != nil {
		return nil, err
	}

	consumer.nsqConsumer = nsqConsumer

	nsqConsumer.AddHandler(nsq.HandlerFunc(
		func(message *nsq.Message) error {
			event, err := NewEvent(message)
			if err != nil {
				return err
			}

			channel <- event
			return nil
		}))

	return consumer, nil
}

// Connecting to NSQD and NSQLookupd is pretty inelegant in the underlying
// library. Let's just be honest about what we're doing and expose that
// functionality directly to users of the messaging library until we can
// come up with a better idea. That being said, prefer that users connect
// to nsqlookupd is, I think, an okay strategy.
func (c *Consumer) ConnectToNSQD(uri *url.URL) {
	c.nsqConsumer.ConnectToNSQD(uri.Host)
}

func (c *Consumer) ConnectToNSQLookupd(uri *url.URL) {
	c.nsqConsumer.ConnectToNSQLookupd(uri.Host)
}

func (c *Consumer) Channel() <-chan EventInterface {
	return c.channel
}

// Stop will first attempt to gracefully shutdown the Consumer. Failing
// to shutdown within a 5-second timeout, it closes channels and shuts down
// the consumer.
func (c *Consumer) Stop() error {
	c.nsqConsumer.Stop()

	var err error
	select {
	case <-c.nsqConsumer.StopChan:
		err = nil
	case <-time.After(5 * time.Second):
		err = fmt.Errorf("Timed out waiting for Consumer to stop.")
	}

	close(c.channel)
	return err
}
