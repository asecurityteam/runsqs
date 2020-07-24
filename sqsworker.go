package runsqs

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

var mutex = &sync.Mutex{}

// DefaultSQSQueueConsumer is a naive sqs queue worker
type DefaultSQSQueueConsumer struct {
	Queue           sqsiface.SQSAPI
	QueueURL        string
	deactivate      chan bool
	MessageConsumer SQSMessageConsumer
}

// StartConsuming starts consuming from the configured SQS queue
func (m *DefaultSQSQueueConsumer) StartConsuming(ctx context.Context) error {

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
				fmt.Println(fmt.Sprintf("Unretryable or Throttle error on sqs retrieval: %v", e.Error()))
			}
			time.Sleep(1 * time.Second)
			continue
		}
		for _, message := range result.Messages {
			_ = m.GetSQSMessageConsumer().ConsumeMessage(ctx, []byte(*message.Body))
			m.ackMessage(ctx, message)
		}
		time.Sleep(time.Duration(1) * time.Millisecond)
	}
}

// StopConsuming stops this DefaultSQSQueueConsumer consuming from the SQS queue
func (m *DefaultSQSQueueConsumer) StopConsuming(ctx context.Context) error {
	mutex.Lock()
	if m.deactivate != nil {
		m.deactivate <- true
	}
	mutex.Unlock()
	return nil
}

// GetSQSMessageConsumer returns the MessageConsumer field. This function implies that
// DefaultSQSQueueConsumer MUST have a MessageConsumer defined.
func (m *DefaultSQSQueueConsumer) GetSQSMessageConsumer() SQSMessageConsumer {
	return m.MessageConsumer
}

func (m *DefaultSQSQueueConsumer) ackMessage(ctx context.Context, message *sqs.Message) {
	for {
		var _, e = m.Queue.DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      aws.String(m.QueueURL),
			ReceiptHandle: message.ReceiptHandle,
		})
		if e != nil {
			if !(request.IsErrorRetryable(e) || request.IsErrorThrottle(e)) {
				fmt.Println(fmt.Sprintf("Unretryable or Throttle error on sqs ack: %v", e.Error()))
				break
			}
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
}
