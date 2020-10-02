package runsqs

import (
	"context"

	"github.com/aws/aws-sdk-go/service/sqs"
)

// SQSConsumer is an interface that represents an aws sqs queue worker.
// Implementers of SQSConsumer are responsible for:
// - SQS connectivity
// - Start and Stop consumption
// - error handling
type SQSConsumer interface {
	StartConsuming(ctx context.Context) error
	StopConsuming(ctx context.Context) error
	GetSQSMessageConsumer() SQSMessageConsumer
}

// SQSMessageConsumer is an interface that defines how a message
// should be consumer. Users are responsible for unmarshalling messages themselves,
// and returning errors.
type SQSMessageConsumer interface {
	ConsumeMessage(ctx context.Context, message *sqs.Message) error
}

// SQSProducer is an interface for producing messages to an aws sqs instance.
// Implementors are responsible for placing messages on an sqs, and also:
// - SQS connectivity
// - error handling
type SQSProducer interface {
	ProduceMessage(message []byte) error
}
