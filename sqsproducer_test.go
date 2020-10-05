package runsqs

import (
	"errors"
	"testing"

	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDefaultSQSProducer_ProduceMessage_Success(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockQueue := NewMockSQSAPI(ctrl)
	producer := &DefaultSQSProducer{
		Queue:    mockQueue,
		QueueURL: "www.queueurl.com",
	}
	message := []byte("a message")
	mockQueue.EXPECT().SendMessage(&sqs.SendMessageInput{
		QueueUrl:    aws.String(producer.QueueURL),
		MessageBody: aws.String(string(message)),
	}).Return(&sqs.SendMessageOutput{}, nil)
	err := producer.ProduceMessage(message)
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
	message := []byte("a message")
	mockQueue.EXPECT().SendMessage(&sqs.SendMessageInput{
		QueueUrl:    aws.String(producer.QueueURL),
		MessageBody: aws.String(string(message)),
	}).Return(&sqs.SendMessageOutput{}, errors.New("error"))
	err := producer.ProduceMessage(message)
	assert.NotNil(t, err)
}
