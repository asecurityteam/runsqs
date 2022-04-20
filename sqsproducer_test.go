package runsqs

import (
	"context"
	"errors"
	"testing"

	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDefaultSQSProducer_QueueUrl_Success(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSAPI(ctrl)
	producer := &DefaultSQSProducer{
		Queue:    mockQueue,
		QueueURL: "www.queueurl.com",
	}
	assert.Equal(t, "www.queueurl.com", producer.QueueUrl())
}

func TestDefaultSQSProducer_ProduceMessage_Success(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSAPI(ctrl)
	producer := &DefaultSQSProducer{
		Queue:    mockQueue,
		QueueURL: "www.queueurl.com",
	}
	sqsMessageInput := &sqs.SendMessageInput{}
	mockQueue.EXPECT().SendMessage(&sqs.SendMessageInput{
		QueueUrl: aws.String(producer.QueueURL),
	}).Return(&sqs.SendMessageOutput{}, nil)
	err := producer.ProduceMessage(context.Background(), sqsMessageInput)
	assert.Nil(t, err)
}

func TestDefaultSQSProducer_ProduceMessage_Failure(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSAPI(ctrl)
	producer := &DefaultSQSProducer{
		Queue:    mockQueue,
		QueueURL: "www.queueurl.com",
	}

	sqsMessageInput := &sqs.SendMessageInput{}
	mockQueue.EXPECT().SendMessage(&sqs.SendMessageInput{
		QueueUrl: aws.String(producer.QueueURL),
	}).Return(&sqs.SendMessageOutput{}, errors.New("error"))
	err := producer.ProduceMessage(context.Background(), sqsMessageInput)
	assert.NotNil(t, err)
}
