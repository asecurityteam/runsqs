package runsqs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultSQSProducerConfig(t *testing.T) {
	component := NewDefaultSQSProducerComponent()
	config := component.Settings()
	consumer, err := component.New(context.Background(), config)
	assert.NotNil(t, consumer)
	assert.Nil(t, err)

}

func TestDefaultSQSProducerConfig_Name(t *testing.T) {
	component := NewDefaultSQSProducerComponent()
	config := component.Settings()
	assert.Equal(t, config.Name(), "sqsproducer")
}
