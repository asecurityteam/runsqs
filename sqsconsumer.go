package runsqs

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

var mutex = &sync.Mutex{}

// DefaultSQSQueueConsumer is a naive implementation of an SQSConsumer.
// This implementation has no support for retries on nonpermanent failures;
// the result of every message consumption is followed by a deletion of
// the message. Furthermore, this implementation does not support concurrent
// processing of messages; messages are processed sequentially.
type DefaultSQSQueueConsumer struct {
	Queue           sqsiface.SQSAPI
	LogFn           LogFn
	QueueURL        string
	deactivate      chan bool
	MessageConsumer SQSMessageConsumer
}

// StartConsuming starts consuming from the configured SQS queue
func (m *DefaultSQSQueueConsumer) StartConsuming(ctx context.Context) error {
	logger := m.LogFn(ctx)

	mutex.Lock()
	m.deactivate = make(chan bool)
	mutex.Unlock()

	var done = ctx.Done()
	for {
		select {
		case <-done:
			return nil
		case <-m.deactivate:
			return nil
		default:
		}
		var result, e = m.Queue.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl: aws.String(m.QueueURL),
			AttributeNames: aws.StringSlice([]string{
				"SentTimestamp",
			}),
			MessageAttributeNames: aws.StringSlice([]string{
				"All",
			}),
			WaitTimeSeconds: aws.Int64(int64(math.Ceil((15 * time.Second).Seconds()))),
		})
		if e != nil {
			if !(request.IsErrorRetryable(e) || request.IsErrorThrottle(e)) {
				logger.Error(e.Error())
			}
			time.Sleep(1 * time.Second)
			continue
		}
		for _, message := range result.Messages {
			_ = m.GetSQSMessageConsumer().ConsumeMessage(ctx, message)
			m.ackMessage(ctx, func() error {
				var _, e = m.Queue.DeleteMessage(&sqs.DeleteMessageInput{
					QueueUrl:      aws.String(m.QueueURL),
					ReceiptHandle: message.ReceiptHandle,
				})
				return e
			})
		}
		time.Sleep(time.Duration(1) * time.Millisecond)
	}
}

// StopConsuming stops this DefaultSQSQueueConsumer consuming from the SQS queue
func (m *DefaultSQSQueueConsumer) StopConsuming(ctx context.Context) error {
	mutex.Lock()
	if m.deactivate != nil {
		close(m.deactivate)
	}
	mutex.Unlock()
	return nil
}

// GetSQSMessageConsumer returns the MessageConsumer field. This function implies that
// DefaultSQSQueueConsumer MUST have a MessageConsumer defined.
func (m *DefaultSQSQueueConsumer) GetSQSMessageConsumer() SQSMessageConsumer {
	return m.MessageConsumer
}

func (m *DefaultSQSQueueConsumer) ackMessage(ctx context.Context, ack func() error) {
	logger := m.LogFn(ctx)

	for {
		e := ack()
		if e != nil {
			if !(request.IsErrorRetryable(e) || request.IsErrorThrottle(e)) {
				logger.Error(e.Error())
				break
			}
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
}

// SmartSQSConsumer is an implementation of an SQSConsumer.
// This implementation supports...
// - retryable and non-retryable errors.
// - a maximum number of retries to be placed on a retryable sqs message
// - concurrent workers
type SmartSQSConsumer struct {
	Queue               sqsiface.SQSAPI
	LogFn               LogFn
	QueueURL            string
	deactivate          chan bool
	MessageConsumer     SQSMessageConsumer
	NumWorkers          uint64
	MessagePoolSize     uint64
	MaxNumberOfMessages uint64
	MaxRetries          uint64
}

// StartConsuming starts consuming from the configured SQS queue
func (m *SmartSQSConsumer) StartConsuming(ctx context.Context) error {
	logger := m.LogFn(ctx)

	mutex.Lock()
	m.deactivate = make(chan bool)
	// messagePool represents a queue of messages that are waiting to be consumed
	messagePool := make(chan *sqs.Message, m.MessagePoolSize)

	// initialize all workers, pass in the pool of messages for each worker
	// to consume from
	for i := uint64(0); i < m.NumWorkers; i++ {
		go m.worker(ctx, messagePool)
	}
	mutex.Unlock()
	var done = ctx.Done()
	for {
		select {
		case <-done:
			// these close statements will cause all workers to eventually terminate
			close(messagePool)
			return nil
		case <-m.deactivate:
			close(messagePool)
			return nil
		default:
		}
		var result, e = m.Queue.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl: aws.String(m.QueueURL),
			AttributeNames: aws.StringSlice([]string{
				"SentTimestamp",
				"ApproximateReceiveCount",
			}),
			MessageAttributeNames: aws.StringSlice([]string{
				"All",
			}),
			MaxNumberOfMessages: aws.Int64(int64(m.MaxNumberOfMessages)),
			WaitTimeSeconds:     aws.Int64(int64(math.Ceil((15 * time.Second).Seconds()))),
		})
		if e != nil {
			if !(request.IsErrorRetryable(e) || request.IsErrorThrottle(e)) {
				logger.Error(e.Error())
			}
			time.Sleep(1 * time.Second)
			continue
		}
		// loop through every message, and queue each message onto messagePool.
		// Because messagePool is a fixed buffered channel, there is potential for this to block.
		// It's important to set MessagePoolSize to a high enough size to account for high sqs throughput
		for _, message := range result.Messages {
			messagePool <- message
		}
		time.Sleep(time.Duration(1) * time.Millisecond)
	}
}

// worker function represents a single "message worker." worker will infinitely process messages until
// messages is closed. worker is responsible for handling deletion of messages, or handling
// messages that have retryable error. On the event of a retryable error, worker will change
// the visibilitytimeout of a message to be the configured retryableErr.VisibilityTimeout
func (m *SmartSQSConsumer) worker(ctx context.Context, messages <-chan *sqs.Message) {
	for message := range messages {
		consumerErr := m.GetSQSMessageConsumer().ConsumeMessage(ctx, message)
		if consumerErr != nil {
			if consumerErr.IsRetryable() {
				// check to see if we've reached the maximum number of retries allowed
				receiveCount := getApproximateReceiveCount(message)
				if receiveCount > m.MaxRetries {
					m.GetSQSMessageConsumer().DeadLetter(ctx, message)
					m.ackMessage(ctx, func() error {
						return m.deleteMessage(message)
					})
					continue
				}
				// retry this message by changing visibility timeout of message
				m.ackMessage(ctx, func() error {
					return m.changeMessageVisibility(message, consumerErr.RetryAfter())
				})
				continue
			}
		}
		// delete message if no error, or error is a permanent, non-retryable error
		m.ackMessage(ctx, func() error {
			return m.deleteMessage(message)
		})
	}
}

// getApproximateReceiveCount is a helper function for retrieving the ApproximateReceiveCount
// Attribute of an SQS message
func getApproximateReceiveCount(message *sqs.Message) uint64 {
	receiveCountString := *(message.Attributes["ApproximateReceiveCount"])
	receiveCount, _ := strconv.ParseInt(receiveCountString, 10, 64)
	return uint64(receiveCount)
}

// deleteMessage is a helper method for deletion of an SQS message
func (m *SmartSQSConsumer) deleteMessage(message *sqs.Message) error {
	var _, e = m.Queue.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(m.QueueURL),
		ReceiptHandle: message.ReceiptHandle,
	})
	return e
}

// changeMessageVisibility is a helper method for changing message visibility of an SQS message
func (m *SmartSQSConsumer) changeMessageVisibility(message *sqs.Message, timeout int64) error {
	var _, e = m.Queue.ChangeMessageVisibility(&sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(m.QueueURL),
		ReceiptHandle:     message.ReceiptHandle,
		VisibilityTimeout: &timeout,
	})
	return e
}

// ackMessage handles acknowledgement of sqs messages. This method takes in an ack callback,
// which when called, handles the specific action to be placed on an sqs message
func (m *SmartSQSConsumer) ackMessage(ctx context.Context, ack func() error) {
	logger := m.LogFn(ctx)

	for {
		e := ack()
		if e != nil {
			if !(request.IsErrorRetryable(e) || request.IsErrorThrottle(e)) {
				logger.Error(e.Error())
				break
			}
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
}

// GetSQSMessageConsumer returns the MessageConsumer field. This function implies that
// DefaultSQSQueueConsumer MUST have a MessageConsumer defined.
func (m *SmartSQSConsumer) GetSQSMessageConsumer() SQSMessageConsumer {
	return m.MessageConsumer
}

// StopConsuming stops this DefaultSQSQueueConsumer consuming from the SQS queue
func (m *SmartSQSConsumer) StopConsuming(ctx context.Context) error {
	mutex.Lock()
	if m.deactivate != nil {
		close(m.deactivate)
	}
	mutex.Unlock()
	return nil
}

// ThroughputSQSConsumer is an implementation of an SQSConsumer.
// This implementation supports...
// - retryable and non-retryable errors.
// - a maximum number of retries to be placed on a retryable sqs message
// - concurrent workers
type ThroughputSQSConsumer struct {
	Queue                    sqsiface.SQSAPI
	LogFn                    LogFn
	QueueURL                 string
	deactivate               chan bool
	MessageConsumer          SQSMessageConsumer
	NumWorkers               uint64
	NumMessageReceiveWorkers uint64
	MessagePoolSize          uint64
	MaxNumberOfMessages      uint64
	MaxRetries               uint64
}

// StartConsuming starts consuming from the configured SQS queue
func (m *ThroughputSQSConsumer) StartConsuming(ctx context.Context) error {
	mutex.Lock()
	m.deactivate = make(chan bool)
	// messagePool represents a queue of messages that are waiting to be consumed
	messagePool := make(chan *sqs.Message, m.MessagePoolSize)

	// initialize all workers, pass in the pool of messages for each worker
	// to consume from
	for i := uint64(0); i < m.NumWorkers; i++ {
		go m.worker(ctx, messagePool)
	}
	// initialize all workers, pass in the pool of messages for each worker
	// to consume from
	for i := uint64(0); i < m.NumWorkers; i++ {
		go m.messageReceiveWorker(ctx, messagePool, i)
	}
	mutex.Unlock()

	return nil

}

// worker function represents a single "message worker." worker will infinitely process messages until
// messages is closed. worker is responsible for handling deletion of messages, or handling
// messages that have retryable error. On the event of a retryable error, worker will change
// the visibilitytimeout of a message to be the configured retryableErr.VisibilityTimeout
func (m *ThroughputSQSConsumer) messageReceiveWorker(ctx context.Context, messagePool chan *sqs.Message, workerNum uint64) error {
	logger := m.LogFn(ctx)

	var done = ctx.Done()
	for {
		select {
		case <-done:
			// these close statements will cause all workers to eventually terminate
			close(messagePool)
			fmt.Println("done signal")
			return nil
		case <-m.deactivate:
			close(messagePool)
			fmt.Println("deactivate signal")
			return nil
		default:
		}
		var result, e = m.Queue.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl: aws.String(m.QueueURL),
			AttributeNames: aws.StringSlice([]string{
				"SentTimestamp",
				"ApproximateReceiveCount",
			}),
			MessageAttributeNames: aws.StringSlice([]string{
				"All",
			}),
			MaxNumberOfMessages: aws.Int64(int64(m.MaxNumberOfMessages)),
			WaitTimeSeconds:     aws.Int64(int64(math.Ceil((15 * time.Second).Seconds()))),
		})
		if e != nil {
			if !(request.IsErrorRetryable(e) || request.IsErrorThrottle(e)) {
				logger.Error(e.Error())
			}
			time.Sleep(1 * time.Second)
			continue
		}
		// loop through every message, and queue each message onto messagePool.
		// Because messagePool is a fixed buffered channel, there is potential for this to block.
		// It's important to set MessagePoolSize to a high enough size to account for high sqs throughput

		for _, message := range result.Messages {
			messagePool <- message
		}
		time.Sleep(time.Duration(1) * time.Millisecond)
	}
}

// worker function represents a single "message worker." worker will infinitely process messages until
// messages is closed. worker is responsible for handling deletion of messages, or handling
// messages that have retryable error. On the event of a retryable error, worker will change
// the visibilitytimeout of a message to be the configured retryableErr.VisibilityTimeout
func (m *ThroughputSQSConsumer) worker(ctx context.Context, messages <-chan *sqs.Message) {
	for message := range messages {
		consumerErr := m.GetSQSMessageConsumer().ConsumeMessage(ctx, message)
		if consumerErr != nil {
			if consumerErr.IsRetryable() {
				// check to see if we've reached the maximum number of retries allowed
				receiveCount := getApproximateReceiveCount(message)
				if receiveCount > m.MaxRetries {
					m.GetSQSMessageConsumer().DeadLetter(ctx, message)
					m.ackMessage(ctx, func() error {
						return m.deleteMessage(message)
					})
					continue
				}
				// retry this message by changing visibility timeout of message
				m.ackMessage(ctx, func() error {
					return m.changeMessageVisibility(message, consumerErr.RetryAfter())
				})
				continue
			}
		}
		// delete message if no error, or error is a permanent, non-retryable error
		m.ackMessage(ctx, func() error {
			return m.deleteMessage(message)
		})
	}
}

// deleteMessage is a helper method for deletion of an SQS message
func (m *ThroughputSQSConsumer) deleteMessage(message *sqs.Message) error {
	var _, e = m.Queue.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(m.QueueURL),
		ReceiptHandle: message.ReceiptHandle,
	})
	return e
}

// changeMessageVisibility is a helper method for changing message visibility of an SQS message
func (m *ThroughputSQSConsumer) changeMessageVisibility(message *sqs.Message, timeout int64) error {
	var _, e = m.Queue.ChangeMessageVisibility(&sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(m.QueueURL),
		ReceiptHandle:     message.ReceiptHandle,
		VisibilityTimeout: &timeout,
	})
	return e
}

// ackMessage handles acknowledgement of sqs messages. This method takes in an ack callback,
// which when called, handles the specific action to be placed on an sqs message
func (m *ThroughputSQSConsumer) ackMessage(ctx context.Context, ack func() error) {
	logger := m.LogFn(ctx)

	for {
		e := ack()
		if e != nil {
			if !(request.IsErrorRetryable(e) || request.IsErrorThrottle(e)) {
				logger.Error(e.Error())
				break
			}
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
}

// GetSQSMessageConsumer returns the MessageConsumer field. This function implies that
// DefaultSQSQueueConsumer MUST have a MessageConsumer defined.
func (m *ThroughputSQSConsumer) GetSQSMessageConsumer() SQSMessageConsumer {
	return m.MessageConsumer
}

// StopConsuming stops this DefaultSQSQueueConsumer consuming from the SQS queue
func (m *ThroughputSQSConsumer) StopConsuming(ctx context.Context) error {
	mutex.Lock()
	if m.deactivate != nil {
		close(m.deactivate)
	}
	mutex.Unlock()
	return nil
}
