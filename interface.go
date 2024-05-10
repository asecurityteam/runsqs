package runsqs

import (
	"context"

	// TODO: switch to latest
	updatedSqs "github.com/aws/aws-sdk-go-v2/service/sqs"
)

// // SQSConsumer is an interface that represents an aws sqs queue worker.
// // Implementers of SQSConsumer are responsible for:
// // - SQS connectivity
// // - Start and Stop consumption
// // - error handling
// type SQSConsumer interface {
// 	StartConsuming(ctx context.Context) error
// 	StopConsuming(ctx context.Context) error
// 	GetSQSMessageConsumer() SQSMessageConsumer
// }

// // SQSMessageConsumer is an interface that defines how a message
// // should be consumer. Users are responsible for unmarshalling messages themselves,
// // and returning errors.
// type SQSMessageConsumer interface {
// 	ConsumeMessage(ctx context.Context, message *sqs.Message) SQSMessageConsumerError
// 	// DeadLetter will be called when MaxRetries is exhausted, only in the SmartSQSConsumer
// 	DeadLetter(ctx context.Context, message *sqs.Message)
// }

// SQSMessageConsumerError represents an error that can be used to indicate to the consumer that an error should be retried.
// Note: RetryAfter should be expressed in seconds
type SQSMessageConsumerError interface {
	IsRetryable() bool
	Error() string
	RetryAfter() int64
}

// TODO: reword this comment
// SQSProducer is an interface for producing messages to an aws sqs instance.
// Implementors are responsible for placing messages on an sqs, and also:
// - SQS connectivity
// - error handling
// - constructing the input *sqs.SendMessageInput
type SQSProducer interface {
	SendMessageBatch(ctx context.Context, params *updatedSqs.SendMessageBatchInput, optFns ...func(*updatedSqs.Options)) (*updatedSqs.SendMessageBatchOutput, error)
	SendMessage(ctx context.Context, params *updatedSqs.SendMessageInput, optFns ...func(*updatedSqs.Options)) (*updatedSqs.SendMessageOutput, error)
}
