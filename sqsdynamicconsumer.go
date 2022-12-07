package runsqs

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/google/uuid"
)

var pollWorkerWaitGroup = &sync.WaitGroup{}
var movingAverage = ewma.NewMovingAverage(1)

// DynamicSQSConsumer is an implementation of an SQSConsumer.
// This implementation supports...
// - retryable and non-retryable errors.
// - a maximum number of retries to be placed on a retryable sqs message
// - concurrent workers
type DynamicSQSQueueConsumer struct {
	Queue           sqsiface.SQSAPI
	LogFn           LogFn
	QueueURL        string
	deactivate      chan bool
	MessageConsumer SQSMessageConsumer
	// MessageTracker           MessageWorker
	NumWorkers               uint64
	NumMessageReceiveWorkers uint64
	MessagePoolSize          uint64
	MaxNumberOfMessages      uint64
	MaxNumPollWorkers        uint64
	MaxRetries               uint64
}

// StartConsuming starts consuming from the configured SQS queue
func (m *DynamicSQSQueueConsumer) StartConsuming(ctx context.Context) error {
	mutex.Lock()
	m.deactivate = make(chan bool)
	// messagePool represents a queue of messages that are waiting to be consumed
	messagePool := make(chan *sqs.Message, m.MessagePoolSize)

	// initialize moderator routine
	go m.moderator(ctx, messagePool)

	// initialize poll worker manager that will control and scale in and scale out of poll workers
	go m.pollWorkerManager(ctx, messagePool)
	// initialize all message workers

	// initialize all workers, pass in the pool of messages for each worker
	// to consume from
	for i := uint64(0); i < m.NumWorkers; i++ {
		go m.worker(ctx, messagePool)
	}

	mutex.Unlock()

	return nil

}

func (m *DynamicSQSQueueConsumer) moderator(ctx context.Context, messagePool chan *sqs.Message) {
	done := ctx.Done()

	select {
	case <-done:
		// wait for senders to close
		pollWorkerWaitGroup.Wait()
		close(messagePool)
		return
	case <-m.deactivate:
		// wait for senders to close
		pollWorkerWaitGroup.Wait()
		close(messagePool)
		return
	}
}

func (m *DynamicSQSQueueConsumer) pollWorkerManager(ctx context.Context, messagePool chan *sqs.Message) {
	logger := m.LogFn(ctx)
	stopSlices := make([]chan bool, 0)

	// start first poll worker
	m.createNewPollWorker(ctx, &stopSlices, messagePool)

	done := ctx.Done()
	scalingTicker := time.NewTicker(60 * time.Second)

	for {
		select {
		case <-done:
			scalingTicker.Stop()
			logger.Info("Poll worker manager received done signal ... exiting")
			return
		case <-m.deactivate:
			scalingTicker.Stop()
			logger.Info("Poll worker manager received done signal ... exiting")
			return
		case <-scalingTicker.C:
			logger.Info(fmt.Sprintln("Determining if pollWorkerManager should scale out or in"))
			if m.determinePollScaleout() {
				// scale up
				logger.Info("pollWorkerManager scaling up poll workers")
				m.createNewPollWorker(ctx, &stopSlices, messagePool)
			} else {
				// scale down
				logger.Info("pollWorkerManager scaling down")

				if len(stopSlices) == 0 {
					continue
				}
				close(stopSlices[0])
				stopSlices = stopSlices[1:]
			}
		}
	}

}

// worker function represents a single "message worker." worker will infinitely process messages until
// messages is closed. worker is responsible for handling deletion of messages, or handling
// messages that have retryable error. On the event of a retryable error, worker will change
// the visibilitytimeout of a message to be the configured retryableErr.VisibilityTimeout
func (m *DynamicSQSQueueConsumer) pollWorker(ctx context.Context, messagePool chan *sqs.Message, stopCh chan bool) error {
	logger := m.LogFn(ctx)
	workerID := uuid.New()

	pollWorkerWaitGroup.Add(1)
	defer pollWorkerWaitGroup.Done()

	var done = ctx.Done()
	logger.Info(fmt.Sprintf("Poll worker %s started polling sqs", workerID.String()))
	for {
		select {
		case <-done:
			logger.Info(fmt.Sprintf("Poll worker %s received done signal ... closing", workerID.String()))
			return nil
		case <-m.deactivate:
			logger.Info(fmt.Sprintf("Poll worker %s received deactivate signal ... closing", workerID.String()))
			return nil
		case <-stopCh:
			logger.Info(fmt.Sprintf("Poll worker %s received stopCH signal ... closing", workerID.String()))
			return nil
		default:
		}

		sqsPollStartTime := time.Now()
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

		logger.Info(fmt.Sprintf("Poll worker %s polled for %s and received %d messages", workerID.String(), time.Since(sqsPollStartTime), len(result.Messages)))
		movingAverage.Add(float64(len(result.Messages)))
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
func (m *DynamicSQSQueueConsumer) worker(ctx context.Context, messages <-chan *sqs.Message) {
	for message := range messages {
		// con, err := m.MessageTracker.GetOrPutMessage(ctx, *message.MessageId)
		// if err != nil {
		// 	fmt.Println(err)
		// 	continue
		// }
		// if !con {
		// 	fmt.Println("message already found")
		// 	continue
		// }

		consumerErr := m.GetSQSMessageConsumer().ConsumeMessage(ctx, message)
		if consumerErr != nil {
			if consumerErr.IsRetryable() {
				// check to see if we've reached the maximum number of retries allowed
				receiveCount := getApproximateReceiveCount(message)
				if receiveCount > m.MaxRetries {
					m.GetSQSMessageConsumer().DeadLetter(ctx, message)
					m.ackMessage(ctx, func() error {
						// if err := m.MessageTracker.UpdateMessageStatus(ctx, *message.MessageId, time.Now(), WaitingToRetry); err != nil {
						// 	return err
						// }
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
			// if err := m.MessageTracker.UpdateMessageStatus(ctx, *message.MessageId, time.Now(), Completed); err != nil {
			// 	return err
			// }
			return m.deleteMessage(message)
		})
	}
}

func (m *DynamicSQSQueueConsumer) determinePollScaleout() bool {
	return movingAverage.Value() > 8
}

// func (m *DynamicSQSQueueConsumer) addSnapshot(snapshot pollSnapshot) {
// 	snapshotMutex.Lock()

// 	if len(snapshots) > 99 {
// 		fmt.Println("dropping first snapshot")
// 		snapshots = snapshots[1:] // pop from front of slice
// 	}
// 	snapshots = append(snapshots, snapshot)

// 	snapshotMutex.Unlock()
// }

func (m *DynamicSQSQueueConsumer) createNewPollWorker(ctx context.Context, stopSlices *[]chan bool, messagePool chan *sqs.Message) {
	stopCh := make(chan bool)
	*stopSlices = append(*stopSlices, stopCh)

	go m.pollWorker(ctx, messagePool, stopCh)
}

// deleteMessage is a helper method for deletion of an SQS message
func (m *DynamicSQSQueueConsumer) deleteMessage(message *sqs.Message) error {
	var _, e = m.Queue.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(m.QueueURL),
		ReceiptHandle: message.ReceiptHandle,
	})
	return e
}

// changeMessageVisibility is a helper method for changing message visibility of an SQS message
func (m *DynamicSQSQueueConsumer) changeMessageVisibility(message *sqs.Message, timeout int64) error {
	var _, e = m.Queue.ChangeMessageVisibility(&sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(m.QueueURL),
		ReceiptHandle:     message.ReceiptHandle,
		VisibilityTimeout: &timeout,
	})
	return e
}

// ackMessage handles acknowledgement of sqs messages. This method takes in an ack callback,
// which when called, handles the specific action to be placed on an sqs message
func (m *DynamicSQSQueueConsumer) ackMessage(ctx context.Context, ack func() error) {
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
func (m *DynamicSQSQueueConsumer) GetSQSMessageConsumer() SQSMessageConsumer {
	return m.MessageConsumer
}

// StopConsuming stops this DefaultSQSQueueConsumer consuming from the SQS queue
func (m *DynamicSQSQueueConsumer) StopConsuming(ctx context.Context) error {
	mutex.Lock()
	if m.deactivate != nil {
		close(m.deactivate)
	}
	mutex.Unlock()
	return nil
}
