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
	return "defaultsqsworker"
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

// New creates a configured MicrosLifecycleEventHandler
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
