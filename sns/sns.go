package sns

import (
	"encoding/json"
	"github.com/BlueDragonX/beacon/beacon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/pkg/errors"
)

// New creates an SNS backend that queues events in the SNS `topic` which lives
// in `region`.
func New(region, topic string) *SNS {
	return NewWithEndpoint("", region, topic)
}

// NewWithEndpoint works like New but allows you to override the AWS HTTP
// endpoint to send requests to.
func NewWithEndpoint(endpoint, region, topic string) *SNS {
	cfg := &aws.Config{}
	if endpoint != "" {
		cfg.Endpoint = aws.String(endpoint)
	}
	if region != "" {
		cfg.Region = aws.String(region)
	}
	return &SNS{
		client: sns.New(session.New(), cfg),
		topic:  topic,
	}
}

// SNS sends container events to an AWS SNS topic. Events are serialized as
// JSON.
type SNS struct {
	client *sns.SNS
	topic  string
}

// ProcessEvent serializes an event in JSON and sends it to the configured SNS
// topic.
func (s *SNS) ProcessEvent(event *beacon.Event) error {
	message, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "failed to serialize event")
	}

	out, err := s.client.Publish(&sns.PublishInput{
		Message:  aws.String(string(message)),
		TopicArn: aws.String(s.topic),
	})
	if err != nil {
		return errors.Wrap(err, "failed to publish event")
	} else if out.MessageId == nil || aws.StringValue(out.MessageId) == "" {
		return errors.New("failed to publish event: no message id returned")
	}
	return nil
}

// Close is a noop for SNS.
func (s *SNS) Close() error {
	return nil
}
