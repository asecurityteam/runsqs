package runsqs

// Decorator is a named type for any function that takes a SQSMessageConsumer and
// returns a SQSMessageConsumer.
type Decorator func(SQSMessageConsumer) SQSMessageConsumer

// Chain is an ordered collection of Decorators.
type Chain []Decorator

// Apply wraps the given SQSMessageConsumer with the Decorator chain.
func (c Chain) Apply(base SQSMessageConsumer) SQSMessageConsumer {
	for x := len(c) - 1; x >= 0; x = x - 1 {
		base = c[x](base)
	}
	return base
}
