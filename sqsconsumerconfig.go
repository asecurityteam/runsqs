package runsqs

import (
	"context"
	"net/http"

	log "github.com/asecurityteam/component-log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// DefaultSQSQueueConsumerConfig represents the configuration to configure DefaultSQSQueueConsumer
type DefaultSQSQueueConsumerConfig struct {
	AWSEndpoint string
	QueueURL    string
	QueueRegion string
	Logger      *log.Config
}

// Name of the configuration
func (*DefaultSQSQueueConsumerConfig) Name() string {
	return "sqsworker"
}

// DefaultSQSQueueConsumerComponent enables creating configured Component
type DefaultSQSQueueConsumerComponent struct {
	Logger *log.Component
}

// NewDefaultSQSQueueConsumerComponent generates a new DefaultSQSQueueConsumerComponent
func NewDefaultSQSQueueConsumerComponent() *DefaultSQSQueueConsumerComponent {
	return &DefaultSQSQueueConsumerComponent{
		Logger: log.NewComponent(),
	}
}

// Settings generates the default configuration for DefaultSQSQueueConsumerComponent
func (c *DefaultSQSQueueConsumerComponent) Settings() *DefaultSQSQueueConsumerConfig {
	return &DefaultSQSQueueConsumerConfig{
		Logger: c.Logger.Settings(),
	}
}

// New creates a configured DefaultSQSQueueConsumer
func (c *DefaultSQSQueueConsumerComponent) New(ctx context.Context, config *DefaultSQSQueueConsumerConfig) (DefaultSQSQueueConsumer, error) {
	var sesh = session.Must(session.NewSession())
	q := sqs.New(sesh, &aws.Config{
		Region:     aws.String(config.QueueRegion),
		HTTPClient: http.DefaultClient,
		Endpoint:   aws.String(config.AWSEndpoint),
	})

	logger, err := c.Logger.New(ctx, config.Logger)
	if err != nil {
		return DefaultSQSQueueConsumer{}, err
	}

	return DefaultSQSQueueConsumer{
		Logger:   logger,
		QueueURL: config.QueueURL,
		Queue:    q,
	}, nil
}

// SmartSQSQueueConsumerConfig represents the configuration to configure SmartSQSQueueConsumer
type SmartSQSQueueConsumerConfig struct {
	AWSEndpoint     string
	QueueURL        string
	QueueRegion     string
	Logger          *log.Config
	NumWorkers      uint64
	MessagePoolSize uint64
}

// Name of the configuration
func (*SmartSQSQueueConsumerConfig) Name() string {
	return "sqsworker"
}

// SmartSQSQueueConsumerComponent enables creating configured Component
type SmartSQSQueueConsumerComponent struct {
	Logger          *log.Component
	NumWorkers      uint64
	MessagePoolSize uint64
}

// NewSmartSQSQueueConsumerComponent generates a new SmartSQSQueueConsumerComponent
func NewSmartSQSQueueConsumerComponent() *SmartSQSQueueConsumerComponent {
	return &SmartSQSQueueConsumerComponent{
		Logger: log.NewComponent(),
	}
}

// Settings generates the default configuration for DefaultSQSQueueConsumerComponent
func (c *SmartSQSQueueConsumerComponent) Settings() *SmartSQSQueueConsumerConfig {
	return &SmartSQSQueueConsumerConfig{
		Logger: c.Logger.Settings(),
	}
}

// New creates a configured SmartSQSConsumer
func (c *SmartSQSQueueConsumerComponent) New(ctx context.Context, config *SmartSQSQueueConsumerConfig) (SmartSQSConsumer, error) {
	var sesh = session.Must(session.NewSession())
	q := sqs.New(sesh, &aws.Config{
		Region:     aws.String(config.QueueRegion),
		HTTPClient: http.DefaultClient,
		Endpoint:   aws.String(config.AWSEndpoint),
	})

	logger, err := c.Logger.New(ctx, config.Logger)
	if err != nil {
		return SmartSQSConsumer{}, err
	}

	return SmartSQSConsumer{
		Logger:          logger,
		QueueURL:        config.QueueURL,
		Queue:           q,
		NumWorkers:      config.NumWorkers,
		MessagePoolSize: config.MessagePoolSize,
	}, nil
}
