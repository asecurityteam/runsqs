package runsqs

import (
	"context"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// DefaultSQSProducerConfig represents the configuration to configure DefaultSQSProducer
type DefaultSQSProducerConfig struct {
	// TODO: do we not need this endpoint config field anymore?
	AWSEndpoint string
	QueueURL    string
	QueueRegion string
}

// Name of the configuration
func (*DefaultSQSProducerConfig) Name() string {
	return "sqsproducer"
}

// DefaultSQSProducerComponent enables creating configured Component
type DefaultSQSProducerComponent struct {
}

// NewDefaultSQSProducerComponent generates a new DefaultSQSQueueConsumerComponent
func NewDefaultSQSProducerComponent() *DefaultSQSProducerComponent {
	return &DefaultSQSProducerComponent{}
}

// Settings generates the default configuration for DefaultSQSProducerComponent
func (c *DefaultSQSProducerComponent) Settings() *DefaultSQSProducerConfig {
	return &DefaultSQSProducerConfig{}
}

// New creates a configured DefaultSQSQueueConsumer
func (c *DefaultSQSProducerComponent) New(ctx context.Context, config *DefaultSQSProducerConfig) (DefaultSQSProducer, error) {

	q := sqs.NewFromConfig(aws.Config{
		Region:     config.QueueRegion,
		HTTPClient: http.DefaultClient,
		// TODO: do we not need this endpoint config field anymore?
		// Endpoint:   config.AWSEndpoint,
	})

	return DefaultSQSProducer{
		queueURL: config.QueueURL,
		Queue:    q,
	}, nil
}
