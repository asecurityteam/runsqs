package runsqs

import (
	"context"
	"net/http"

	cfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// DefaultSQSProducerConfig represents the configuration to configure DefaultSQSProducer
type DefaultSQSProducerConfig struct {
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
	sqsConfig, err := cfg.LoadDefaultConfig(
		ctx,
		cfg.WithRegion(config.QueueRegion),
		cfg.WithHTTPClient(http.DefaultClient),
		cfg.WithBaseEndpoint(config.AWSEndpoint),
	)
	if err != nil {
		return DefaultSQSProducer{}, err
	}

	q := sqs.NewFromConfig(sqsConfig)

	return DefaultSQSProducer{
		queueURL: config.QueueURL,
		Queue:    q,
	}, nil
}
