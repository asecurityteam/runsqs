package runsqs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
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
	ConsumeMessage(ctx context.Context, message *types.Message) SQSMessageConsumerError
	// DeadLetter will be called when MaxRetries is exhausted, only in the SmartSQSConsumer
	DeadLetter(ctx context.Context, message *types.Message)
}

// SQSMessageConsumerError represents an error that can be used to indicate to the consumer that an error should be retried.
// Note: RetryAfter should be expressed in seconds
type SQSMessageConsumerError interface {
	IsRetryable() bool
	Error() string
	RetryAfter() int32
}

// SQSProducer is an interface for producing messages to an aws sqs instance.
// Implementors are responsible for placing messages on an sqs, and also:
// - SQS connectivity
// - error handling
// - constructing the input *sqs.SendMessageInput
type SQSProducer interface {
	QueueURL() string
	BatchProduceMessage(ctx context.Context, messageBatchInput *sqs.SendMessageBatchInput) (*sqs.SendMessageBatchOutput, error)
	ProduceMessage(ctx context.Context, messageInput *sqs.SendMessageInput) error
}

// SQSClient is an interface for interacting with the actual SQS resource in AWS
type SQSClient interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	SendMessageBatch(ctx context.Context, params *sqs.SendMessageBatchInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageBatchOutput, error)
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
	ChangeMessageVisibility(ctx context.Context, params *sqs.ChangeMessageVisibilityInput, optFns ...func(*sqs.Options)) (*sqs.ChangeMessageVisibilityOutput, error)
}
