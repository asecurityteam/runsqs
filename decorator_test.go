package runsqs

import (
	"testing"

	"github.com/golang/mock/gomock"
	"gotest.tools/assert"
)

func TestConsumerDecorator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	counter := 0

	mockSQSMessageConsumer1 := NewMockSQSMessageConsumer(ctrl)
	mockSQSMessageConsumer2 := NewMockSQSMessageConsumer(ctrl)

	mockSQSMessageConsumerBase := NewMockSQSMessageConsumer(ctrl)

	decorator1 := func(SQSMessageConsumer) SQSMessageConsumer {
		// these assertions asserts the order of decorators applied
		assert.Equal(t, counter, 0)
		counter++
		return mockSQSMessageConsumer1
	}

	decorator2 := func(SQSMessageConsumer) SQSMessageConsumer {
		assert.Equal(t, counter, 1)
		return mockSQSMessageConsumer2
	}

	chain := ConsumerChain([]ConsumerDecorator{decorator2, decorator1})

	chain.Apply(mockSQSMessageConsumerBase)

}

func TestProducerDecorator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	counter := 0

	producer1 := NewMockSQSProducer(ctrl)
	producer2 := NewMockSQSProducer(ctrl)

	base := NewMockSQSProducer(ctrl)

	decorator1 := func(SQSProducer) SQSProducer {
		// these assertions asserts the order of decorators applied
		assert.Equal(t, counter, 0)
		counter++
		return producer1
	}

	decorator2 := func(SQSProducer) SQSProducer {
		assert.Equal(t, counter, 1)
		return producer2
	}

	chain := ProducerChain([]ProducerDecorator{decorator2, decorator1})

	chain.Apply(base)

}
