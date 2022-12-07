package runsqs

import (
	"context"
	"sync"
	"testing"

	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
	gomock "github.com/golang/mock/gomock"
)

/********************************
TestSmartSQSConsumer TESTS
********************************/

// TestSmartSQSConsumer_GoldenPath tests whether it can retrieve 5000 messages,
// process those 5000 messages concurrently, and delete those 5000 messages
func TestDynamicSQSConsumer_GoldenPath(t *testing.T) {
	// mocks
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSAPI(ctrl)
	mockLogger := NewMockLogger(ctrl)
	mockMessageConsumer := NewMockSQSMessageConsumer(ctrl)

	// testBlocker is used to make this test deterministic(avoid timeouts)
	var testBlocker sync.WaitGroup
	var consumer = &DynamicSQSQueueConsumer{
		LogFn:                    func(context.Context) Logger { return mockLogger },
		QueueURL:                 queueURL,
		Queue:                    mockQueue,
		MessageConsumer:          mockMessageConsumer,
		NumWorkers:               10,
		MessagePoolSize:          100,
		MaxNumberOfMessages:      1,
		MaxRetries:               1,
		NumMessageReceiveWorkers: 5,
		MaxNumPollWorkers:        5,
	}

	messages := []*sqs.Message{}
	for i := 0; i < 1000; i++ {
		messages = append(messages, defaultSQSMessage)
	}

	receiveMessageOutput := &sqs.ReceiveMessageOutput{
		Messages: messages,
	}
	// the following mocks test for exactly 5 successful message consumptions, no more no less
	mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()
	mockQueue.EXPECT().ReceiveMessage(sqsInputWithReceiveCount).Return(receiveMessageOutput, nil).Times(5)
	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), defaultSQSMessage).Return(nil).Times(5000)
	mockQueue.EXPECT().DeleteMessage(gomock.Any()).DoAndReturn(func(interface{}) (*sqs.DeleteMessageOutput, error) {
		testBlocker.Done()
		return nil, nil
	}).Times(5000)

	// infinitely ping empty sqs
	mockQueue.EXPECT().ReceiveMessage(sqsInputWithReceiveCount).Return(sqsEmptyMessageOutput, nil).AnyTimes()

	testBlocker.Add(5000)
	consumer.StartConsuming(context.Background()) // nolint
	testBlocker.Wait()
	consumer.StopConsuming(context.Background()) // nolint

}

// TestSmartSQSConsumer_ReceivingMessageFailure tests whether it can retrieve 2 messages, both of them fail,
// but only one is retryable. The non retryable causes a log.error
func TestDynamicSQSConsumer_ReceivingMessageFailure(t *testing.T) {
	// mocks
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSAPI(ctrl)
	mockLogger := NewMockLogger(ctrl)
	mockMessageConsumer := NewMockSQSMessageConsumer(ctrl)

	// testBlocker is used to make this test deterministic(avoid timeouts)
	var testBlocker sync.WaitGroup
	var consumer = &DynamicSQSQueueConsumer{
		LogFn:               func(context.Context) Logger { return mockLogger },
		QueueURL:            queueURL,
		Queue:               mockQueue,
		MessageConsumer:     mockMessageConsumer,
		NumWorkers:          10,
		MessagePoolSize:     100,
		MaxRetries:          1,
		MaxNumberOfMessages: 1,
	}

	// 1 retryables, 1 error log
	mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()
	mockQueue.EXPECT().ReceiveMessage(sqsInputWithReceiveCount).Return(sqsEmptyMessageOutput, awserr.New("RequestThrottled", "test", nil))
	mockLogger.EXPECT().Error(gomock.Any()).Times(1)
	// non retryable
	mockQueue.EXPECT().ReceiveMessage(sqsInputWithReceiveCount).Return(sqsEmptyMessageOutput, awserr.New("RequestCanceled", "test", nil))

	// infinitely ping empty sqs
	mockQueue.EXPECT().ReceiveMessage(sqsInputWithReceiveCount).DoAndReturn(func(interface{}) (interface{}, error) {
		defer testBlocker.Done()
		return sqsEmptyMessageOutput, nil
	}).AnyTimes()
	testBlocker.Add(1)
	go consumer.StartConsuming(context.Background()) // nolint
	testBlocker.Wait()
	consumer.StopConsuming(context.Background()) // nolint

}

// // TestSmartSQSConsumer_ConsumeMessageFailures tests retryable and nonretryable ConsumeMessage
// // errors.
func TestDynamicSQSConsumer_ConsumeMessageFailures(t *testing.T) {
	// mocks
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSAPI(ctrl)
	mockLogger := NewMockLogger(ctrl)
	mockMessageConsumer := NewMockSQSMessageConsumer(ctrl)
	mockSQSMessageConsumerError := NewMockSQSMessageConsumerError(ctrl)

	// testBlocker is used to make this test deterministic(avoid timeouts)
	var testBlocker sync.WaitGroup
	var consumer = &DynamicSQSQueueConsumer{
		LogFn:               func(context.Context) Logger { return mockLogger },
		QueueURL:            queueURL,
		Queue:               mockQueue,
		MessageConsumer:     mockMessageConsumer,
		NumWorkers:          1,
		MessagePoolSize:     1,
		MaxRetries:          1,
		MaxNumberOfMessages: 1,
	}
	firstReceiveCount := "1"
	firstSQSMessage := &sqs.Message{
		Body:          aws.String("body"),
		ReceiptHandle: aws.String("receipt"),
		Attributes: map[string]*string{
			"ApproximateReceiveCount": &firstReceiveCount,
		},
	}
	messages1 := []*sqs.Message{}
	for i := 0; i < 1; i++ {
		messages1 = append(messages1, firstSQSMessage)
	}

	receiveMessageOutput1 := &sqs.ReceiveMessageOutput{
		Messages: messages1,
	}

	secondReceiveCount := "1"
	secondSQSMessage := &sqs.Message{
		Body:          aws.String("body"),
		ReceiptHandle: aws.String("receipt"),
		Attributes: map[string]*string{
			"ApproximateReceiveCount": &secondReceiveCount,
		},
	}
	messages2 := []*sqs.Message{
		secondSQSMessage,
	}

	receiveMessageOutput2 := &sqs.ReceiveMessageOutput{
		Messages: messages2,
	}

	mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()

	mockQueue.EXPECT().ReceiveMessage(sqsInputWithReceiveCount).Return(receiveMessageOutput1, nil)
	mockQueue.EXPECT().ReceiveMessage(sqsInputWithReceiveCount).Return(receiveMessageOutput2, nil)

	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), firstSQSMessage).Return(mockSQSMessageConsumerError)
	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), secondSQSMessage).Return(mockSQSMessageConsumerError)

	mockSQSMessageConsumerError.EXPECT().IsRetryable().Return(true)
	mockSQSMessageConsumerError.EXPECT().RetryAfter().Return(int64(3))

	mockQueue.EXPECT().ChangeMessageVisibility(gomock.Any()).DoAndReturn(func(interface{}) (*sqs.ChangeMessageVisibilityOutput, error) {
		testBlocker.Done()
		return nil, nil
	})
	mockSQSMessageConsumerError.EXPECT().IsRetryable().Return(false)

	mockQueue.EXPECT().DeleteMessage(gomock.Any()).DoAndReturn(func(interface{}) (*sqs.DeleteMessageOutput, error) {
		testBlocker.Done()
		return nil, nil
	})

	testBlocker.Add(2)
	go consumer.StartConsuming(context.Background()) // nolint
	testBlocker.Wait()
	consumer.StopConsuming(context.Background()) // nolint

}

// TestSmartSQSConsumer_ConsumeMessageFailures tests a retryable sqs message until it hits
// max retries
func TestDynamicSQSConsumer_MaxRetries(t *testing.T) {
	// mocks
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSAPI(ctrl)
	mockLogger := NewMockLogger(ctrl)
	mockMessageConsumer := NewMockSQSMessageConsumer(ctrl)
	mockSQSMessageConsumerError := NewMockSQSMessageConsumerError(ctrl)

	// testBlocker is used to make this test deterministic(avoid timeouts)
	var testBlocker sync.WaitGroup
	var consumer = &DynamicSQSQueueConsumer{
		LogFn:               func(context.Context) Logger { return mockLogger },
		QueueURL:            queueURL,
		Queue:               mockQueue,
		MessageConsumer:     mockMessageConsumer,
		NumWorkers:          1,
		MessagePoolSize:     1,
		MaxRetries:          2,
		MaxNumberOfMessages: 1,
	}
	firstReceiveCount := "1"
	firstSQSMessage := &sqs.Message{
		Body:          aws.String("body"),
		ReceiptHandle: aws.String("receipt"),
		Attributes: map[string]*string{
			"ApproximateReceiveCount": &firstReceiveCount,
		},
	}
	messages1 := []*sqs.Message{}
	for i := 0; i < 1; i++ {
		messages1 = append(messages1, firstSQSMessage)
	}

	receiveMessageOutput1 := &sqs.ReceiveMessageOutput{
		Messages: messages1,
	}

	secondReceiveCount := "2"
	secondSQSMessage := &sqs.Message{
		Body:          aws.String("body"),
		ReceiptHandle: aws.String("receipt"),
		Attributes: map[string]*string{
			"ApproximateReceiveCount": &secondReceiveCount,
		},
	}
	messages2 := []*sqs.Message{
		secondSQSMessage,
	}

	receiveMessageOutput2 := &sqs.ReceiveMessageOutput{
		Messages: messages2,
	}

	thirdReceiveCount := "3"
	thirdSQSMessage := &sqs.Message{
		Body:          aws.String("body"),
		ReceiptHandle: aws.String("receipt"),
		Attributes: map[string]*string{
			"ApproximateReceiveCount": &thirdReceiveCount,
		},
	}
	messages3 := []*sqs.Message{
		thirdSQSMessage,
	}

	receiveMessageOutput3 := &sqs.ReceiveMessageOutput{
		Messages: messages3,
	}

	mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()

	mockQueue.EXPECT().ReceiveMessage(sqsInputWithReceiveCount).Return(receiveMessageOutput1, nil)
	mockQueue.EXPECT().ReceiveMessage(sqsInputWithReceiveCount).Return(receiveMessageOutput2, nil)
	mockQueue.EXPECT().ReceiveMessage(sqsInputWithReceiveCount).Return(receiveMessageOutput3, nil)

	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), firstSQSMessage).Return(mockSQSMessageConsumerError)
	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), secondSQSMessage).Return(mockSQSMessageConsumerError)
	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), thirdSQSMessage).Return(mockSQSMessageConsumerError)
	mockMessageConsumer.EXPECT().DeadLetter(gomock.Any(), thirdSQSMessage)

	mockSQSMessageConsumerError.EXPECT().IsRetryable().Return(true).Times(3)
	mockSQSMessageConsumerError.EXPECT().RetryAfter().Return(int64(3)).Times(2)

	mockQueue.EXPECT().ChangeMessageVisibility(gomock.Any()).DoAndReturn(func(interface{}) (*sqs.ChangeMessageVisibilityOutput, error) {
		testBlocker.Done()
		return nil, nil
	})

	mockQueue.EXPECT().ChangeMessageVisibility(gomock.Any()).DoAndReturn(func(interface{}) (*sqs.ChangeMessageVisibilityOutput, error) {
		testBlocker.Done()
		return nil, nil
	})

	mockQueue.EXPECT().DeleteMessage(gomock.Any()).DoAndReturn(func(interface{}) (*sqs.DeleteMessageOutput, error) {
		testBlocker.Done()
		return nil, nil
	})

	testBlocker.Add(3)
	go consumer.StartConsuming(context.Background()) // nolint
	testBlocker.Wait()
	consumer.StopConsuming(context.Background()) // nolint

}

// TestSmartSQSConsumer_ConsumeMessageAckFailure tests retryable acks.
func TestDynamicSQSConsumer_ConsumeMessageAckFailure(t *testing.T) {
	// mocks
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSAPI(ctrl)
	mockLogger := NewMockLogger(ctrl)
	mockMessageConsumer := NewMockSQSMessageConsumer(ctrl)

	// testBlocker is used to make this test deterministic(avoid timeouts)
	var testBlocker sync.WaitGroup
	var consumer = &DynamicSQSQueueConsumer{
		LogFn:               func(context.Context) Logger { return mockLogger },
		QueueURL:            queueURL,
		Queue:               mockQueue,
		MessageConsumer:     mockMessageConsumer,
		NumWorkers:          10,
		MessagePoolSize:     100,
		MaxRetries:          1,
		MaxNumberOfMessages: 1,
	}

	messages := []*sqs.Message{}
	for i := 0; i < 1; i++ {
		messages = append(messages, defaultSQSMessage)
	}

	receiveMessageOutput := &sqs.ReceiveMessageOutput{
		Messages: messages,
	}

	mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()

	mockQueue.EXPECT().ReceiveMessage(sqsInputWithReceiveCount).Return(receiveMessageOutput, nil)

	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), defaultSQSMessage).Return(nil)

	mockQueue.EXPECT().DeleteMessage(gomock.Any()).DoAndReturn(func(interface{}) (*sqs.DeleteMessageOutput, error) {
		return nil, awserr.New("RequestThrottled", "test", nil)
	}).Times(1)
	mockQueue.EXPECT().DeleteMessage(gomock.Any()).DoAndReturn(func(interface{}) (*sqs.DeleteMessageOutput, error) {
		testBlocker.Done()
		return nil, nil
	}).Times(1)
	mockQueue.EXPECT().ReceiveMessage(sqsInputWithReceiveCount).Return(sqsEmptyMessageOutput, nil).AnyTimes()
	testBlocker.Add(1)
	go consumer.StartConsuming(context.Background()) // nolint
	testBlocker.Wait()
	consumer.StopConsuming(context.Background()) // nolint

}
