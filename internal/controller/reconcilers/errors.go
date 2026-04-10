package reconcilers

// KafkaError wraps an error that originates from Kafka-specific configuration
// (TLS certificates, SASL credentials). Use errors.As to detect it at the
// top-level Reconcile and set an appropriate status reason.
type KafkaError struct{ Err error }

func (e *KafkaError) Error() string { return e.Err.Error() }
func (e *KafkaError) Unwrap() error { return e.Err }

func WrapKafkaError(err error) error {
	if err == nil {
		return nil
	}
	return &KafkaError{Err: err}
}
