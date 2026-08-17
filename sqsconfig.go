package runsqs

import (
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
)

func newSQSClientConfig(region, endpoint string) (*session.Session, *aws.Config) {
	sesh := session.Must(session.NewSession())

	config := &aws.Config{
		Region:     aws.String(region),
		HTTPClient: http.DefaultClient,
	}
	if sesh.Config.UseFIPSEndpoint != endpoints.FIPSEndpointStateEnabled {
		config.Endpoint = aws.String(endpoint)
	}

	return sesh, config
}
