package tgo

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

// Consumer holds all infromation
// about the RabbitMQ connection
// This setup does limit a consumer
// to one exchange. This should not be
// an issue. Having to connect to multiple
// exchanges means something else is
// structured improperly.
type Consumer struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	done         chan error
	consumerTag  string // Name that consumer identifies itself to the server with
	uri          string // uri of the rabbitmq server
	vhost        string //vhost
	exchange     string // exchange that we will bind to
	exchangeType string // topic, direct, etc...
	bindingKey   string // routing key that we are using
}

// NewConsumer returns a Consumer struct
// that has been initialized properly
// essentially don't touch conn, channel, or
// done and you can create Consumer manually
func NewConsumer() *Consumer {
	return &Consumer{
		consumerTag:  globalConfig.RabbitMQ.ConsumerTag,
		uri:          globalConfig.RabbitMQ.Uri + globalConfig.RabbitMQ.Vhost,
		exchange:     globalConfig.RabbitMQ.Exchange,
		exchangeType: globalConfig.RabbitMQ.ExchangeType,
		done:         make(chan error),
	}

}

// ReConnect is called in places where NotifyClose() channel is called
// wait 30 seconds before trying to reconnect. Any shorter amount of time
// will  likely destroy the error log while waiting for servers to come
// back online. This requires two parameters which is just to satisfy
// the AccounceQueue call and allows greater flexability
func (c *Consumer) ReConnect(queueName, bindingKey string) (<-chan amqp.Delivery, error) {
	time.Sleep(30 * time.Second)

	if err := c.Connect(); err != nil {
		log.Printf("Could not connect in reconnect call: %v", err.Error())
	}

	deliveries, err := c.AnnounceQueue(queueName, bindingKey)
	if err != nil {
		return deliveries, errors.New("Couldn't connect")
	}

	return deliveries, nil
}

// Connect to RabbitMQ server
func (c *Consumer) Connect() error {

	var err error

	log.Printf("dialing %q", c.uri)
	c.conn, err = amqp.Dial(c.uri)
	if err != nil {
		return fmt.Errorf("Dial: %s", err)
	}

	go func() {
		// Waits here for the channel to be closed
		log.Printf("closing: %s", <-c.conn.NotifyClose(make(chan *amqp.Error)))
		// Let Handle know it's not time to reconnect
		c.done <- errors.New("Channel Closed")
	}()

	log.Printf("got Connection, getting Channel")
	c.channel, err = c.conn.Channel()
	if err != nil {
		return fmt.Errorf("Channel: %s", err)
	}

	log.Printf("got Channel, declaring Exchange (%q)", c.exchange)
	if err = c.channel.ExchangeDeclare(
		c.exchange,     // name of the exchange
		c.exchangeType, // type
		true,           // durable
		false,          // delete when complete
		false,          // internal
		false,          // noWait
		nil,            // arguments
	); err != nil {
		return fmt.Errorf("Exchange Declare: %s", err)
	}

	return nil
}

// AnnounceQueue sets the queue that will be listened to for this
// connection...
func (c *Consumer) AnnounceQueue(queueName, bindingKey string) (<-chan amqp.Delivery, error) {
	log.Printf("declared Exchange, declaring Queue %q", queueName)
	queue, err := c.channel.QueueDeclare(
		queueName, // name of the queue
		true,      // durable
		false,     // delete when usused
		false,     // exclusive
		false,     // noWait
		nil,       // arguments
	)

	if err != nil {
		return nil, fmt.Errorf("Queue Declare: %s", err)
	}

	log.Printf("declared Queue (%q %d messages, %d consumers), binding to Exchange (key %q)",
		queue.Name, queue.Messages, queue.Consumers, bindingKey)

	// Qos determines the amount of messages that the queue will pass to you before
	// it waits for you to ack them. This will slow down queue consumption but
	// give you more certainty that all messages are being processed. As load increases
	// I would reccomend upping the about of Threads and Processors the go process
	// uses before changing this although you will eventually need to reach some
	// balance between threads, procs, and Qos.
	err = c.channel.Qos(50, 0, false)
	if err != nil {
		return nil, fmt.Errorf("Error setting qos: %s", err)
	}

	if err = c.channel.QueueBind(
		queue.Name, // name of the queue
		bindingKey, // bindingKey
		c.exchange, // sourceExchange
		false,      // noWait
		nil,        // arguments
	); err != nil {
		return nil, fmt.Errorf("Queue Bind: %s", err)
	}

	log.Printf("Queue bound to Exchange, starting Consume (consumer tag %q)", c.consumerTag)
	deliveries, err := c.channel.Consume(
		queue.Name,    // name
		c.consumerTag, // consumerTag,
		false,         // noAck
		false,         // exclusive
		false,         // noLocal
		false,         // noWait
		nil,           // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("Queue Consume: %s", err)
	}

	return deliveries, nil
}

// Handle has all the logic to make sure your program keeps running
// d should be a delievey channel as created when you call AnnounceQueue
// fn should be a function that handles the processing of deliveries
// this should be the last thing called in main as code under it will
// become unreachable unless put int a goroutine. The q and rk params
// are redundent but allow you to have multiple queue listeners in main
// without them you would be tied into only using one queue per connection
func (c *Consumer) Handle(
	d <-chan amqp.Delivery,
	fn func(<-chan amqp.Delivery),
	threads int,
	queue string,
	routingKey string) {

	var err error

	for {
		for i := 0; i < threads; i++ {
			go fn(d)
		}

		// Go into reconnect loop when
		// c.done is passed non nil values
		if <-c.done != nil {
			d, err = c.ReConnect(queue, routingKey)
			if err != nil {
				// Very likely chance of failing
				// should not cause worker to terminate
				log.Fatalf("Reconnecting Error: %s", err)
			}
		}
		log.Println("Reconnected... possibly")
	}
}
