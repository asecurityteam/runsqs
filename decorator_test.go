package runsqs

import (
	"testing"

	"github.com/golang/mock/gomock"
	"gotest.tools/assert"
)

func TestDecorator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	counter := 0

	mockSQSMessageConsumer1 := NewMockSQSMessageConsumer(ctrl)
	mockSQSMessageConsumer2 := NewMockSQSMessageConsumer(ctrl)

	mockSQSMessageConsumerBase := NewMockSQSMessageConsumer(ctrl)

	decorator1 := func(SQSMessageConsumer) SQSMessageConsumer {
		assert.Equal(t, counter, 0)
		counter += 1
		return mockSQSMessageConsumer1
	}

	decorator2 := func(SQSMessageConsumer) SQSMessageConsumer {
		assert.Equal(t, counter, 1)
		return mockSQSMessageConsumer2
	}

	chain := Chain([]Decorator{decorator2, decorator1})

	chain.Apply(mockSQSMessageConsumerBase)

}
