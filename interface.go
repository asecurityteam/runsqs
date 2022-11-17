package runsqs

import (
	"context"
	"time"

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
	ConsumeMessage(ctx context.Context, message *sqs.Message) SQSMessageConsumerError
	// DeadLetter will be called when MaxRetries is exhausted, only in the SmartSQSConsumer
	DeadLetter(ctx context.Context, message *sqs.Message)
}

// SQSMessageConsumerError represents an error that can be used to indicate to the consumer that an error should be retried.
// Note: RetryAfter should be expressed in seconds
type SQSMessageConsumerError interface {
	IsRetryable() bool
	Error() string
	RetryAfter() int64
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

// MessageTracker placeholder
type MessageTrackerBackend interface {
	GetMessage(ctx context.Context, id string) (*SQSMessage, error)
	PutNewMessage(ctx context.Context, message *SQSMessage) error
	UpdateMessageStatus(ctx context.Context, id string, updated_at time.Time, status MessageStatus) error
}

type MessageTracker interface {
	GetOrPutMessage(ctx context.Context, id string) (bool, error)
	UpdateMessageStatus(ctx context.Context, id string, updated_at time.Time, status MessageStatus) error
}

type MessageStatus string

const (
	Processing     MessageStatus = "processing"
	Failed         MessageStatus = "failed"
	WaitingToRetry MessageStatus = "waiting_to_retry"
	Completed      MessageStatus = "completed"
)

// SQSMessage placeholder
type SQSMessage struct {
	ID        string        `json:"id"`
	Status    MessageStatus `json:"status"`
	UpdatedAt string        `json:"updated_at"`
	TTL       int64         `json:"ttl,omitempty"`
}
