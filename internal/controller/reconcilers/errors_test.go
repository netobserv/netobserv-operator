package reconcilers

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapKafkaErrorNil(t *testing.T) {
	assert.Nil(t, WrapKafkaError(nil))
}

func TestKafkaErrorDetection(t *testing.T) {
	original := fmt.Errorf("secret not found")
	wrapped := WrapKafkaError(original)

	var ke *KafkaError
	require.True(t, errors.As(wrapped, &ke))
	assert.Equal(t, "secret not found", ke.Error())
	assert.Equal(t, original, errors.Unwrap(wrapped))
}

func TestKafkaErrorThroughWrapping(t *testing.T) {
	original := fmt.Errorf("TLS cert missing")
	wrapped := WrapKafkaError(original)
	rewrapped := fmt.Errorf("processing config: %w", wrapped)

	var ke *KafkaError
	assert.True(t, errors.As(rewrapped, &ke), "errors.As should find KafkaError through fmt.Errorf wrapping")
}

func TestNonKafkaErrorNotDetected(t *testing.T) {
	plain := fmt.Errorf("permission denied")

	var ke *KafkaError
	assert.False(t, errors.As(plain, &ke))
}
