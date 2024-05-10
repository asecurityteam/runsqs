package runsqs

import (
	"context"
	"errors"
	"testing"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDefaultSQSProducer_QueueUrl_Success(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSProducer(ctrl)
	producer := NewDefaultSQSProducer(mockQueue, "www.queueurl.com")
	assert.Equal(t, "www.queueurl.com", producer.QueueURL())
}

func TestDefaultSQSProducer_ProduceMessage_Success(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSProducer(ctrl)
	producer := NewDefaultSQSProducer(mockQueue, "www.queueurl.com")
	ctx := context.Background()

	sqsMessageInput := &sqs.SendMessageInput{}
	mockQueue.EXPECT().SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl: aws.String(producer.QueueURL()),
	}).Return(&sqs.SendMessageOutput{}, nil)
	err := producer.ProduceMessage(context.Background(), sqsMessageInput)
	assert.Nil(t, err)
}

func TestDefaultSQSProducer_ProduceMessage_Failure(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSProducer(ctrl)
	producer := NewDefaultSQSProducer(mockQueue, "www.queueurl.com")
	ctx := context.Background()

	sqsMessageInput := &sqs.SendMessageInput{}
	mockQueue.EXPECT().SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl: aws.String(producer.QueueURL()),
	}).Return(&sqs.SendMessageOutput{}, errors.New("error"))
	err := producer.ProduceMessage(context.Background(), sqsMessageInput)
	assert.NotNil(t, err)
}

func TestDefaultSQSProducer_BatchProduceMessage_Success(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSProducer(ctrl)
	producer := NewDefaultSQSProducer(mockQueue, "www.queueurl.com")
	ctx := context.Background()

	sqsBatchMessageInput := &sqs.SendMessageBatchInput{}
	mockQueue.EXPECT().SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
		QueueUrl: aws.String(producer.QueueURL()),
	}).Return(&sqs.SendMessageBatchOutput{}, nil)
	resp, err := producer.BatchProduceMessage(context.Background(), sqsBatchMessageInput)
	assert.Nil(t, err)
	assert.Len(t, resp.Failed, 0)
}

func TestDefaultSQSProducer_BatchProduceMessage_Failure(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSProducer(ctrl)
	producer := NewDefaultSQSProducer(mockQueue, "www.queueurl.com")
	ctx := context.Background()

	sqsBatchMessageInput := &sqs.SendMessageBatchInput{}
	mockQueue.EXPECT().SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
		QueueUrl: aws.String(producer.QueueURL()),
	}).Return(&sqs.SendMessageBatchOutput{}, errors.New("error"))
	resp, err := producer.BatchProduceMessage(context.Background(), sqsBatchMessageInput)
	assert.NotNil(t, err)
	assert.Len(t, resp.Failed, 0)
}

func TestDefaultSQSProducer_BatchProduceMessage_Failed_Items(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSProducer(ctrl)
	producer := NewDefaultSQSProducer(mockQueue, "www.queueurl.com")
	ctx := context.Background()

	sqsBatchMessageInput := &sqs.SendMessageBatchInput{}
	mockQueue.EXPECT().SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
		QueueUrl: aws.String(producer.QueueURL()),
	}).Return(&sqs.SendMessageBatchOutput{
		Failed: []types.BatchResultErrorEntry{{}},
	}, nil)

	resp, err := producer.BatchProduceMessage(context.Background(), sqsBatchMessageInput)
	assert.Nil(t, err)
	assert.Len(t, resp.Failed, 1)
}
