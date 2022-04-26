package runsqs

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

// DefaultSQSProducer is a basic sqs producer
type DefaultSQSProducer struct {
	Queue    sqsiface.SQSAPI
	queueURL string
}

func NewDefaultSQSProducer(queue sqsiface.SQSAPI, url string) *DefaultSQSProducer {
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
	_, e := producer.Queue.SendMessage(messageInput)
	return e
}
