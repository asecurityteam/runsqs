package runsqs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// DefaultSQSProducer is a basic sqs producer
type DefaultSQSProducer struct {
	Queue    SQSClient
	queueURL string
}

// NewDefaultSQSProducer initializes a new DefaultSQSProducer
func NewDefaultSQSProducer(queue SQSClient, url string) *DefaultSQSProducer {
	return &DefaultSQSProducer{
		Queue:    queue,
		queueURL: url,
	}
}

// QueueURL retrieves the queue URL used by the DefaultSQSProducer
func (producer *DefaultSQSProducer) QueueURL() string {
	return producer.queueURL
}

// ProduceMessage produces a message to the configured sqs queue,
// along with setting the queueURL to use
func (producer *DefaultSQSProducer) ProduceMessage(ctx context.Context, messageInput *sqs.SendMessageInput) error {
	messageInput.QueueUrl = aws.String(producer.QueueURL())
	_, e := producer.Queue.SendMessage(ctx, messageInput)
	return e
}

// BatchProduceMessage produces a batch of messages to the configured sqs queue,
// along with setting the queueURL to use
func (producer *DefaultSQSProducer) BatchProduceMessage(ctx context.Context, messageInput *sqs.SendMessageBatchInput) (*sqs.SendMessageBatchOutput, error) {
	messageInput.QueueUrl = aws.String(producer.QueueURL())
	return producer.Queue.SendMessageBatch(ctx, messageInput)
}
