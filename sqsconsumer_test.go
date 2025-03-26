package runsqs

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var queueURL = "http://awssomething.com"

var sqsInput = &sqs.ReceiveMessageInput{
	QueueUrl: aws.String(queueURL),
	MessageSystemAttributeNames: []types.MessageSystemAttributeName{
		"SentTimestamp",
	},
	MessageAttributeNames: []string{
		"All",
	},
	WaitTimeSeconds: int32(math.Ceil((15 * time.Second).Seconds())),
}

var sqsInputWithReceiveCount = &sqs.ReceiveMessageInput{
	QueueUrl: aws.String(queueURL),
	MessageSystemAttributeNames: []types.MessageSystemAttributeName{
		"SentTimestamp",
		"ApproximateReceiveCount",
	},
	MessageAttributeNames: []string{
		"All",
	},
	MaxNumberOfMessages: int32(1),
	WaitTimeSeconds:     int32(math.Ceil((15 * time.Second).Seconds())),
}

var sqsEmptyMessageOutput = &sqs.ReceiveMessageOutput{
	Messages: []types.Message{},
}

var defaultSQSMessage = types.Message{
	Body:          aws.String("body"),
	ReceiptHandle: aws.String("receipt"),
}

/********************************
TestDefaultSQSQueueConsumer TESTS
********************************/

// TestDefaultSQSQueueConsumer_GoldenPath tests whether it can retrieve 5 messages,
// process those 5 messages, and delete those 5 messages
func TestDefaultSQSQueueConsumer_GoldenPath(t *testing.T) {
	ctx := context.Background()
	// mocks
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSClient(ctrl)
	mockLogger := NewMockLogger(ctrl)
	mockMessageConsumer := NewMockSQSMessageConsumer(ctrl)

	// testBlocker is used to make this test deterministic(avoid timeouts)
	var testBlocker sync.WaitGroup
	var consumer = &DefaultSQSQueueConsumer{
		LogFn:           func(context.Context) Logger { return mockLogger },
		QueueURL:        queueURL,
		retrierConfig:   *retry.NewStandard(),
		Queue:           mockQueue,
		MessageConsumer: mockMessageConsumer,
	}

	receiveMessageOutput := &sqs.ReceiveMessageOutput{
		Messages: []types.Message{
			defaultSQSMessage,
		},
	}

	// the following mocks test for exactly 5 successful message consumptions, no more no less
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInput, nil).Return(receiveMessageOutput, nil).Times(5)
	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), &defaultSQSMessage).Return(nil).Times(5)
	mockQueue.EXPECT().DeleteMessage(ctx, gomock.Any(), nil).DoAndReturn(func(interface{}, interface{}, interface{}) (*sqs.DeleteMessageOutput, error) {
		testBlocker.Done()
		return nil, nil
	}).Times(5)

	// infinitely ping empty sqs
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInput, nil).Return(sqsEmptyMessageOutput, nil).AnyTimes()

	testBlocker.Add(5)
	go consumer.StartConsuming(ctx) // nolint
	testBlocker.Wait()
	err := consumer.StopConsuming(ctx)
	assert.NoError(t, err)
}

// TestDefaultSQSQueueConsumer_ReceivingMessageFailure tests tests whether it can retrieve 2 messages, both of them fail,
// but only one is retryable. The non retryable causes a log.error
func TestDefaultSQSQueueConsumer_ReceivingMessageFailure(t *testing.T) {
	ctx := context.Background()
	// mocks
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSClient(ctrl)
	mockLogger := NewMockLogger(ctrl)
	mockMessageConsumer := NewMockSQSMessageConsumer(ctrl)

	// testBlocker is used to make this test deterministic(avoid timeouts)
	var testBlocker sync.WaitGroup
	var consumer = &DefaultSQSQueueConsumer{
		LogFn:           func(context.Context) Logger { return mockLogger },
		retrierConfig:   *retry.NewStandard(),
		QueueURL:        queueURL,
		Queue:           mockQueue,
		MessageConsumer: mockMessageConsumer,
	}

	// 1 retryables, 1 error log
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInput, nil).Return(sqsEmptyMessageOutput, &smithy.GenericAPIError{
		Code:    "RequestThrottled",
		Message: "test",
		Fault:   0,
	})
	mockLogger.EXPECT().Error(gomock.Any()).Times(1)
	// non retryable
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInput, nil).Return(sqsEmptyMessageOutput, &smithy.GenericAPIError{
		Code:    "RequestCanceled",
		Message: "test",
		Fault:   0,
	})

	// infinitely ping empty sqs
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInput, nil).DoAndReturn(func(interface{}, interface{}, interface{}) (interface{}, error) {
		defer testBlocker.Done()
		return sqsEmptyMessageOutput, nil
	}).AnyTimes()
	testBlocker.Add(1)
	go consumer.StartConsuming(ctx) // nolint
	testBlocker.Wait()
	err := consumer.StopConsuming(ctx)
	assert.NoError(t, err)

}

/********************************
TestSmartSQSConsumer TESTS
********************************/

// TestSmartSQSConsumer_GoldenPath tests whether it can retrieve 5000 messages,
// process those 5000 messages concurrently, and delete those 5000 messages
func TestSmartSQSConsumer_GoldenPath(t *testing.T) {
	ctx := context.Background()
	// mocks
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSClient(ctrl)
	mockLogger := NewMockLogger(ctrl)
	mockMessageConsumer := NewMockSQSMessageConsumer(ctrl)

	// testBlocker is used to make this test deterministic(avoid timeouts)
	var testBlocker sync.WaitGroup
	var consumer = &SmartSQSConsumer{
		LogFn:               func(context.Context) Logger { return mockLogger },
		QueueURL:            queueURL,
		Queue:               mockQueue,
		MessageConsumer:     mockMessageConsumer,
		retrierConfig:       *retry.NewStandard(),
		NumWorkers:          10,
		MessagePoolSize:     100,
		MaxNumberOfMessages: 1,
		MaxRetries:          1,
	}

	messages := []types.Message{}
	for i := 0; i < 1000; i++ {
		messages = append(messages, defaultSQSMessage)
	}

	receiveMessageOutput := &sqs.ReceiveMessageOutput{
		Messages: messages,
	}
	// the following mocks test for exactly 5 successful message consumptions, no more no less
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInputWithReceiveCount, nil).Return(receiveMessageOutput, nil).Times(5)
	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), &defaultSQSMessage).Return(nil).Times(5000)
	mockQueue.EXPECT().DeleteMessage(ctx, gomock.Any(), nil).DoAndReturn(func(interface{}, interface{}, interface{}) (*sqs.DeleteMessageOutput, error) {
		testBlocker.Done()
		return nil, nil
	}).Times(5000)

	// infinitely ping empty sqs
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInputWithReceiveCount, nil).Return(sqsEmptyMessageOutput, nil).AnyTimes()

	testBlocker.Add(5000)
	go consumer.StartConsuming(ctx) // nolint
	testBlocker.Wait()
	err := consumer.StopConsuming(ctx)
	assert.NoError(t, err)
}

// TestSmartSQSConsumer_ReceivingMessageFailure tests whether it can retrieve 2 messages, both of them fail,
// but only one is retryable. The non retryable causes a log.error
func TestSmartSQSConsumer_ReceivingMessageFailure(t *testing.T) {
	ctx := context.Background()
	// mocks
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSClient(ctrl)
	mockLogger := NewMockLogger(ctrl)
	mockMessageConsumer := NewMockSQSMessageConsumer(ctrl)

	// testBlocker is used to make this test deterministic(avoid timeouts)
	var testBlocker sync.WaitGroup
	var consumer = &SmartSQSConsumer{
		LogFn:               func(context.Context) Logger { return mockLogger },
		QueueURL:            queueURL,
		Queue:               mockQueue,
		MessageConsumer:     mockMessageConsumer,
		retrierConfig:       *retry.NewStandard(),
		NumWorkers:          10,
		MessagePoolSize:     100,
		MaxRetries:          1,
		MaxNumberOfMessages: 1,
	}

	// 1 retryables, 1 error log
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInputWithReceiveCount, nil).Return(sqsEmptyMessageOutput, &smithy.GenericAPIError{
		Code:    "RequestThrottled",
		Message: "test",
		Fault:   0,
	})
	mockLogger.EXPECT().Error(gomock.Any()).Times(1)
	// non retryable
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInputWithReceiveCount, nil).Return(sqsEmptyMessageOutput, &smithy.GenericAPIError{
		Code:    "RequestCanceled",
		Message: "test",
		Fault:   0,
	})

	// infinitely ping empty sqs
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInputWithReceiveCount, nil).DoAndReturn(func(interface{}, interface{}, interface{}) (interface{}, error) {
		defer testBlocker.Done()
		return sqsEmptyMessageOutput, nil
	}).AnyTimes()
	testBlocker.Add(1)
	go consumer.StartConsuming(ctx) // nolint
	testBlocker.Wait()
	err := consumer.StopConsuming(ctx)
	assert.NoError(t, err)
}

// TestSmartSQSConsumer_ConsumeMessageFailures tests retryable and nonretryable ConsumeMessage
// errors.
func TestSmartSQSConsumer_ConsumeMessageFailures(t *testing.T) {
	ctx := context.Background()
	// mocks
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSClient(ctrl)
	mockLogger := NewMockLogger(ctrl)
	mockMessageConsumer := NewMockSQSMessageConsumer(ctrl)
	mockSQSMessageConsumerError := NewMockSQSMessageConsumerError(ctrl)

	// testBlocker is used to make this test deterministic(avoid timeouts)
	var testBlocker sync.WaitGroup
	var consumer = &SmartSQSConsumer{
		LogFn:               func(context.Context) Logger { return mockLogger },
		QueueURL:            queueURL,
		Queue:               mockQueue,
		MessageConsumer:     mockMessageConsumer,
		retrierConfig:       *retry.NewStandard(),
		NumWorkers:          1,
		MessagePoolSize:     1,
		MaxRetries:          1,
		MaxNumberOfMessages: 1,
	}
	firstReceiveCount := "1"
	firstSQSMessage := types.Message{
		Body:          aws.String("body"),
		ReceiptHandle: aws.String("receipt"),
		Attributes: map[string]string{
			"ApproximateReceiveCount": firstReceiveCount,
		},
	}
	messages1 := []types.Message{}
	for i := 0; i < 1; i++ {
		messages1 = append(messages1, firstSQSMessage)
	}

	receiveMessageOutput1 := &sqs.ReceiveMessageOutput{
		Messages: messages1,
	}

	secondReceiveCount := "1"
	secondSQSMessage := types.Message{
		Body:          aws.String("body"),
		ReceiptHandle: aws.String("receipt"),
		Attributes: map[string]string{
			"ApproximateReceiveCount": secondReceiveCount,
		},
	}
	messages2 := []types.Message{
		secondSQSMessage,
	}

	receiveMessageOutput2 := &sqs.ReceiveMessageOutput{
		Messages: messages2,
	}

	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInputWithReceiveCount, nil).Return(receiveMessageOutput1, nil)
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInputWithReceiveCount, nil).Return(receiveMessageOutput2, nil)

	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), &firstSQSMessage).Return(mockSQSMessageConsumerError)
	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), &secondSQSMessage).Return(mockSQSMessageConsumerError)

	mockSQSMessageConsumerError.EXPECT().IsRetryable().Return(true)
	mockSQSMessageConsumerError.EXPECT().RetryAfter().Return(int32(3))

	mockQueue.EXPECT().ChangeMessageVisibility(ctx, gomock.Any(), gomock.Any()).DoAndReturn(func(interface{}, interface{}, interface{}) (*sqs.ChangeMessageVisibilityOutput, error) {
		testBlocker.Done()
		return nil, nil
	})
	mockSQSMessageConsumerError.EXPECT().IsRetryable().Return(false)

	mockQueue.EXPECT().DeleteMessage(ctx, gomock.Any(), gomock.Any()).DoAndReturn(func(interface{}, interface{}, interface{}) (*sqs.DeleteMessageOutput, error) {
		testBlocker.Done()
		return nil, nil
	})

	testBlocker.Add(2)
	go consumer.StartConsuming(ctx) // nolint
	testBlocker.Wait()
	err := consumer.StopConsuming(ctx)
	assert.NoError(t, err)
}

// TestSmartSQSConsumer_ConsumeMessageFailures tests a retryable sqs message until it hits
// max retries
func TestSmartSQSConsumer_MaxRetries(t *testing.T) {
	ctx := context.Background()
	// mocks
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSClient(ctrl)
	mockLogger := NewMockLogger(ctrl)
	mockMessageConsumer := NewMockSQSMessageConsumer(ctrl)
	mockSQSMessageConsumerError := NewMockSQSMessageConsumerError(ctrl)

	// testBlocker is used to make this test deterministic(avoid timeouts)
	var testBlocker sync.WaitGroup
	var consumer = &SmartSQSConsumer{
		LogFn:               func(context.Context) Logger { return mockLogger },
		QueueURL:            queueURL,
		Queue:               mockQueue,
		MessageConsumer:     mockMessageConsumer,
		retrierConfig:       *retry.NewStandard(),
		NumWorkers:          1,
		MessagePoolSize:     1,
		MaxRetries:          2,
		MaxNumberOfMessages: 1,
	}
	firstReceiveCount := "1"
	firstSQSMessage := types.Message{
		Body:          aws.String("body"),
		ReceiptHandle: aws.String("receipt"),
		Attributes: map[string]string{
			"ApproximateReceiveCount": firstReceiveCount,
		},
	}
	messages1 := []types.Message{}
	for i := 0; i < 1; i++ {
		messages1 = append(messages1, firstSQSMessage)
	}

	receiveMessageOutput1 := &sqs.ReceiveMessageOutput{
		Messages: messages1,
	}

	secondReceiveCount := "2"
	secondSQSMessage := types.Message{
		Body:          aws.String("body"),
		ReceiptHandle: aws.String("receipt"),
		Attributes: map[string]string{
			"ApproximateReceiveCount": secondReceiveCount,
		},
	}
	messages2 := []types.Message{
		secondSQSMessage,
	}

	receiveMessageOutput2 := &sqs.ReceiveMessageOutput{
		Messages: messages2,
	}

	thirdReceiveCount := "3"
	thirdSQSMessage := types.Message{
		Body:          aws.String("body"),
		ReceiptHandle: aws.String("receipt"),
		Attributes: map[string]string{
			"ApproximateReceiveCount": thirdReceiveCount,
		},
	}
	messages3 := []types.Message{
		thirdSQSMessage,
	}

	receiveMessageOutput3 := &sqs.ReceiveMessageOutput{
		Messages: messages3,
	}

	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInputWithReceiveCount, nil).Return(receiveMessageOutput1, nil)
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInputWithReceiveCount, nil).Return(receiveMessageOutput2, nil)
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInputWithReceiveCount, nil).Return(receiveMessageOutput3, nil)

	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), &firstSQSMessage).Return(mockSQSMessageConsumerError)
	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), &secondSQSMessage).Return(mockSQSMessageConsumerError)
	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), &thirdSQSMessage).Return(mockSQSMessageConsumerError)
	mockMessageConsumer.EXPECT().DeadLetter(gomock.Any(), &thirdSQSMessage)

	mockSQSMessageConsumerError.EXPECT().IsRetryable().Return(true).Times(3)
	mockSQSMessageConsumerError.EXPECT().RetryAfter().Return(int32(3)).Times(2)

	mockQueue.EXPECT().ChangeMessageVisibility(ctx, gomock.Any(), nil).DoAndReturn(func(interface{}, interface{}, interface{}) (*sqs.ChangeMessageVisibilityOutput, error) {
		testBlocker.Done()
		return nil, nil
	})

	mockQueue.EXPECT().ChangeMessageVisibility(ctx, gomock.Any(), nil).DoAndReturn(func(interface{}, interface{}, interface{}) (*sqs.ChangeMessageVisibilityOutput, error) {
		testBlocker.Done()
		return nil, nil
	})

	mockQueue.EXPECT().DeleteMessage(ctx, gomock.Any(), nil).DoAndReturn(func(interface{}, interface{}, interface{}) (*sqs.DeleteMessageOutput, error) {
		testBlocker.Done()
		return nil, nil
	})

	testBlocker.Add(3)
	go consumer.StartConsuming(ctx) // nolint
	testBlocker.Wait()
	err := consumer.StopConsuming(ctx)
	assert.NoError(t, err)
}

// TestSmartSQSConsumer_ConsumeMessageAckFailure tests retryable acks.
func TestSmartSQSConsumer_ConsumeMessageAckFailure(t *testing.T) {
	ctx := context.Background()
	// mocks
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSClient(ctrl)
	mockLogger := NewMockLogger(ctrl)
	mockMessageConsumer := NewMockSQSMessageConsumer(ctrl)

	// testBlocker is used to make this test deterministic(avoid timeouts)
	var testBlocker sync.WaitGroup
	var consumer = &SmartSQSConsumer{
		LogFn:               func(context.Context) Logger { return mockLogger },
		QueueURL:            queueURL,
		Queue:               mockQueue,
		MessageConsumer:     mockMessageConsumer,
		retrierConfig:       *retry.NewStandard(),
		NumWorkers:          10,
		MessagePoolSize:     100,
		MaxRetries:          1,
		MaxNumberOfMessages: 1,
	}

	messages := []types.Message{}
	for i := 0; i < 1; i++ {
		messages = append(messages, defaultSQSMessage)
	}

	receiveMessageOutput := &sqs.ReceiveMessageOutput{
		Messages: messages,
	}
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInputWithReceiveCount, nil).Return(receiveMessageOutput, nil)

	mockMessageConsumer.EXPECT().ConsumeMessage(gomock.Any(), &defaultSQSMessage).Return(nil)

	mockQueue.EXPECT().DeleteMessage(ctx, gomock.Any(), nil).DoAndReturn(func(interface{}, interface{}, interface{}) (*sqs.DeleteMessageOutput, error) {
		return nil, &smithy.GenericAPIError{
			Code:    "RequestThrottled",
			Message: "test",
			Fault:   0,
		}
	}).Times(1)
	mockQueue.EXPECT().DeleteMessage(ctx, gomock.Any(), nil).DoAndReturn(func(interface{}, interface{}, interface{}) (*sqs.DeleteMessageOutput, error) {
		testBlocker.Done()
		return nil, nil
	}).Times(1)
	mockQueue.EXPECT().ReceiveMessage(ctx, sqsInputWithReceiveCount, nil).Return(sqsEmptyMessageOutput, nil).AnyTimes()
	testBlocker.Add(1)
	go consumer.StartConsuming(ctx) // nolint
	testBlocker.Wait()
	err := consumer.StopConsuming(ctx)
	assert.NoError(t, err)
}
