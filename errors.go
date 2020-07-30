package runsqs

// RetryableConsumerError represents a possible error type an SQSMessageConsumer could return
// on a call to ConsumeMessage. Users can set VisibilityTimeout, and implementors of SQSConsumer
// can leverage VisibilityTimeout to change the visibility of an sqs message for retry purposes.
type RetryableConsumerError struct {
	WrappedError      error
	VisibilityTimeout int64
}

// Error implements type error
func (e RetryableConsumerError) Error() string {
	return e.WrappedError.Error()
}
