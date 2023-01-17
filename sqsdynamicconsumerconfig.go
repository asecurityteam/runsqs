package runsqs

import (
	"context"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// DynamicSQSQueueConsumerConfig represents the configuration to configure DynamicSQSQueueConsumer
type DynamicSQSQueueConsumerConfig struct {
	AWSEndpoint              string
	QueueURL                 string
	QueueRegion              string
	NumWorkers               uint64
	NumMessageReceiveWorkers uint64
	MessagePoolSize          uint64
	MaxNumberOfMessages      uint64
	MaxRetries               uint64
}

// Name of the configuration
func (*DynamicSQSQueueConsumerConfig) Name() string {
	return "sqsworker"
}

// DynamicSQSQueueConsumerComponent enables creating configured Component
type DynamicSQSQueueConsumerComponent struct {
}

// NewDynamicSQSQueueConsumerComponent generates a new DynamicSQSQueueConsumerComponent
func NewDynamicSQSQueueConsumerComponent() *DynamicSQSQueueConsumerComponent {
	return &DynamicSQSQueueConsumerComponent{}
}

// Settings generates the default configuration for DefaultDynamicSQSQueueConsumerConfig
func (c *DynamicSQSQueueConsumerComponent) Settings() *DynamicSQSQueueConsumerConfig {
	return &DynamicSQSQueueConsumerConfig{
		NumWorkers:               defaultNumWorkers,
		NumMessageReceiveWorkers: defaultNumReceiveWorkers,
		MessagePoolSize:          defaultMessagePoolSize,
		MaxRetries:               defaultMaxRetries,
		MaxNumberOfMessages:      defaultMaxNumberOfMessages,
	}
}

// New creates a configured DynamicSQSConsumer
func (c *DynamicSQSQueueConsumerComponent) New(ctx context.Context, config *DynamicSQSQueueConsumerConfig) (DynamicSQSQueueConsumer, error) {
	var sesh = session.Must(session.NewSession())
	q := sqs.New(sesh, &aws.Config{
		Region:     aws.String(config.QueueRegion),
		HTTPClient: http.DefaultClient,
		Endpoint:   aws.String(config.AWSEndpoint),
	})

	return DynamicSQSQueueConsumer{
		LogFn:                    LoggerFromContext,
		QueueURL:                 config.QueueURL,
		Queue:                    q,
		NumWorkers:               config.NumWorkers,
		NumMessageReceiveWorkers: config.NumMessageReceiveWorkers,
		MessagePoolSize:          config.MessagePoolSize,
		MaxNumberOfMessages:      config.MaxNumberOfMessages,
		MaxRetries:               config.MaxRetries,
	}, nil
}
