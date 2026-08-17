package runsqs

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
)

func TestNewSQSClientConfig(t *testing.T) {
	const (
		region         = "us-east-1"
		customEndpoint = "https://custom-sqs.example.com"
	)

	t.Run("FIPS omits custom endpoint", func(t *testing.T) {
		t.Setenv("AWS_USE_FIPS_ENDPOINT", "true")

		sesh, config := newSQSClientConfig(region, customEndpoint)

		assert.Nil(t, config.Endpoint)
		client := sqs.New(sesh, config)
		assert.Equal(t, "https://sqs-fips.us-east-1.amazonaws.com", client.ClientInfo.Endpoint)
	})

	t.Run("disabled FIPS preserves custom endpoint", func(t *testing.T) {
		t.Setenv("AWS_USE_FIPS_ENDPOINT", "false")

		_, config := newSQSClientConfig(region, customEndpoint)

		assert.Equal(t, customEndpoint, *config.Endpoint)
	})

	t.Run("unset FIPS preserves custom endpoint", func(t *testing.T) {
		t.Setenv("AWS_USE_FIPS_ENDPOINT", "")

		_, config := newSQSClientConfig(region, customEndpoint)

		assert.Equal(t, customEndpoint, *config.Endpoint)
	})
}
