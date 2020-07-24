package runsqs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

// DefaultSQSProducer is a basic sqs producer
type DefaultSQSProducer struct {
	Queue    sqsiface.SQSAPI
	QueueURL string
}

// ProduceMessage produces a message to the configured sqs queue
func (producer *DefaultSQSProducer) ProduceMessage(message []byte) error {
	_, e := producer.Queue.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    aws.String(producer.QueueURL),
		MessageBody: aws.String(string(message)),
	})
	return e
}
