package runsqs

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	defaultPollInterval        = 1000 // milliseconds
	defaultNumWorkers          = 1
	defaultMessagePoolSize     = 1
	defaultMaxRetries          = 3
	defaultMaxNumberOfMessages = 1
)

// DefaultSQSQueueConsumerConfig represents the configuration to configure DefaultSQSQueueConsumer
type DefaultSQSQueueConsumerConfig struct {
	AWSEndpoint  string
	QueueURL     string
	QueueRegion  string
	PollInterval time.Duration
}

// Name of the configuration
func (*DefaultSQSQueueConsumerConfig) Name() string {
	return "sqsworker"
}

// DefaultSQSQueueConsumerComponent enables creating configured Component
type DefaultSQSQueueConsumerComponent struct {
}

// NewDefaultSQSQueueConsumerComponent generates a new DefaultSQSQueueConsumerComponent
func NewDefaultSQSQueueConsumerComponent() *DefaultSQSQueueConsumerComponent {
	return &DefaultSQSQueueConsumerComponent{}
}

// Settings generates the default configuration for DefaultSQSQueueConsumerComponent
func (c *DefaultSQSQueueConsumerComponent) Settings() *DefaultSQSQueueConsumerConfig {
	return &DefaultSQSQueueConsumerConfig{
		PollInterval: defaultPollInterval * time.Millisecond,
	}
}

// New creates a configured DefaultSQSQueueConsumer
func (c *DefaultSQSQueueConsumerComponent) New(ctx context.Context, config *DefaultSQSQueueConsumerConfig) (DefaultSQSQueueConsumer, error) {
	sesh, sqsConfig := newSQSClientConfig(config.QueueRegion, config.AWSEndpoint)
	q := sqs.New(sesh, sqsConfig)

	return DefaultSQSQueueConsumer{
		LogFn:    LoggerFromContext,
		QueueURL: config.QueueURL,
		Queue:    q,
	}, nil
}

// SmartSQSQueueConsumerConfig represents the configuration to configure SmartSQSQueueConsumer
type SmartSQSQueueConsumerConfig struct {
	AWSEndpoint         string
	QueueURL            string
	QueueRegion         string
	NumWorkers          uint64
	MessagePoolSize     uint64
	MaxNumberOfMessages uint64
	MaxRetries          uint64
	PollInterval        time.Duration
}

// Name of the configuration
func (*SmartSQSQueueConsumerConfig) Name() string {
	return "sqsworker"
}

// SmartSQSQueueConsumerComponent enables creating configured Component
type SmartSQSQueueConsumerComponent struct {
}

// NewSmartSQSQueueConsumerComponent generates a new SmartSQSQueueConsumerComponent
func NewSmartSQSQueueConsumerComponent() *SmartSQSQueueConsumerComponent {
	return &SmartSQSQueueConsumerComponent{}
}

// Settings generates the default configuration for DefaultSQSQueueConsumerComponent
func (c *SmartSQSQueueConsumerComponent) Settings() *SmartSQSQueueConsumerConfig {
	return &SmartSQSQueueConsumerConfig{
		NumWorkers:          defaultNumWorkers,
		MessagePoolSize:     defaultMessagePoolSize,
		MaxRetries:          defaultMaxRetries,
		MaxNumberOfMessages: defaultMaxNumberOfMessages,
		PollInterval:        defaultPollInterval * time.Millisecond,
	}
}

// New creates a configured SmartSQSConsumer
func (c *SmartSQSQueueConsumerComponent) New(ctx context.Context, config *SmartSQSQueueConsumerConfig) (SmartSQSConsumer, error) {
	sesh, sqsConfig := newSQSClientConfig(config.QueueRegion, config.AWSEndpoint)
	q := sqs.New(sesh, sqsConfig)

	return SmartSQSConsumer{
		LogFn:               LoggerFromContext,
		QueueURL:            config.QueueURL,
		Queue:               q,
		NumWorkers:          config.NumWorkers,
		MessagePoolSize:     config.MessagePoolSize,
		MaxNumberOfMessages: config.MaxNumberOfMessages,
		MaxRetries:          config.MaxRetries,
		PollInterval:        config.PollInterval,
	}, nil
}
