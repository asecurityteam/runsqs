package runsqs

// // ConsumerDecorator is a named type for any function that takes a SQSMessageConsumer and
// // returns a SQSMessageConsumer.
// type ConsumerDecorator func(SQSMessageConsumer) SQSMessageConsumer

// // ConsumerChain is an ordered collection of ConsumerDecorator.
// type ConsumerChain []ConsumerDecorator

// // Apply wraps the given SQSMessageConsumer with the Decorator chain.
// func (c ConsumerChain) Apply(base SQSMessageConsumer) SQSMessageConsumer {
// 	for x := len(c) - 1; x >= 0; x = x - 1 {
// 		base = c[x](base)
// 	}
// 	return base
// }

// ProducerDecorator is a named type for any function that takes a SQSProducer and
// returns a SQSProducer.
type ProducerDecorator func(SQSProducer) SQSProducer

// ProducerChain is an ordered collection of Decorators.
type ProducerChain []ProducerDecorator

// Apply wraps the given SQSProducer with the Decorator chain.
func (c ProducerChain) Apply(base SQSProducer) SQSProducer {
	for x := len(c) - 1; x >= 0; x = x - 1 {
		base = c[x](base)
	}
	return base
}
