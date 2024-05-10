package runsqs

// import (
// 	"context"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// )

// func TestNewDefaultSQSQueueConsumerConfig(t *testing.T) {
// 	component := NewDefaultSQSQueueConsumerComponent()
// 	config := component.Settings()
// 	consumer, err := component.New(context.Background(), config)
// 	assert.NotNil(t, consumer)
// 	assert.Nil(t, err)

// }

// func TestDefaultSQSQueueConsumerConfig_Name(t *testing.T) {
// 	component := NewDefaultSQSQueueConsumerComponent()
// 	config := component.Settings()
// 	assert.Equal(t, config.Name(), "sqsworker")
// }

// func TestNewSmartSQSQueueConsumerConfig(t *testing.T) {
// 	component := NewSmartSQSQueueConsumerComponent()
// 	config := component.Settings()
// 	consumer, err := component.New(context.Background(), config)
// 	assert.NotNil(t, consumer)
// 	assert.Nil(t, err)

// }

// func TestSmartSQSQueueConsumerConfig_Name(t *testing.T) {
// 	component := NewSmartSQSQueueConsumerComponent()
// 	config := component.Settings()
// 	assert.Equal(t, config.Name(), "sqsworker")
// }
