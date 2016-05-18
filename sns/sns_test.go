package sns_test

import (
	sns "."
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/BlueDragonX/beacon/beacon"
	"github.com/pkg/errors"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	TEST_REGION = "us-east-1"
	TEST_TOPIC  = "arn:aws:sns:us-east-1:698519295917:TestTopic"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func RandomID() string {
	hexRunes := []rune("abcdef1234567890")
	b := make([]rune, 32)
	for i := range b {
		b[i] = hexRunes[rand.Intn(len(hexRunes))]
	}
	return string(b)
}

func WaitForEvents(ch <-chan *beacon.Event, n int, timeout time.Duration) ([]*beacon.Event, error) {
	events := make([]*beacon.Event, 0, n)
	timer := time.After(timeout)
	for i := 0; i < n; i++ {
		select {
		case event, ok := <-ch:
			if !ok {
				return events, errors.New("channel closed")
			}
			events = append(events, event)
		case <-timer:
			return events, errors.New("timed out")
		}
	}
	return events, nil
}

// NewServer create a new test HTTP server which responds to SNS Publish messages.
func NewServer(t *testing.T, events chan<- *beacon.Event) *httptest.Server {
	handler := func(w http.ResponseWriter, r *http.Request) {
		msg := r.PostFormValue("Message")
		event := &beacon.Event{}

		if err := json.Unmarshal([]byte(msg), event); err == nil {
			if events != nil {
				events <- event
			}

			type PublishResult struct {
				MessageId string
			}

			type ResponseMetadata struct {
				RequestId string
			}

			type PublishResponse struct {
				PublishResult    PublishResult
				ResponseMetadata ResponseMetadata
			}

			response := PublishResponse{
				PublishResult: PublishResult{
					MessageId: RandomID(),
				},
				ResponseMetadata: ResponseMetadata{
					RequestId: RandomID(),
				},
			}

			w.WriteHeader(200)
			xenc := xml.NewEncoder(w)
			if err := xenc.Encode(response); err != nil {
				t.Fatalf("failed to encode xml response: %s", err)
			}
		} else {
			t.Errorf("failed to decode json message: %s\n%s", err, msg)
			w.WriteHeader(400)
			fmt.Fprintf(w, "unable to decode event: %s", err)
		}
	}
	return httptest.NewServer(http.HandlerFunc(handler))
}

func ContainersEqual(a *beacon.Container, b *beacon.Container) error {
	if a.ID != b.ID {
		return errors.Errorf("container.ID inequal: %s != %s", a.ID, b.ID)
	}
	if a.Service != b.Service {
		return errors.Errorf("container.Service inequal: %s != %s", a.Service, b.Service)
	}
	if len(a.Labels) != len(b.Labels) {
		return errors.Errorf("container.Labels inequal length: %d != %d", len(a.Labels), len(b.Labels))
	}
	for k, v1 := range a.Labels {
		if v2, ok := b.Labels[k]; !ok || v1 != v2 {
			return errors.Errorf("container.Labels[%s] inequal: %s != %s", k, v1, v2)
		}
	}
	if len(a.Bindings) != len(b.Bindings) {
		return errors.Errorf("container.bindings have length: %d != %d", len(a.Bindings), len(b.Bindings))
	}
	for n, b1 := range a.Bindings {
		b2 := b.Bindings[n]
		if b1.HostPort != b2.HostPort || b1.ContainerPort != b2.ContainerPort || b1.Protocol != b2.Protocol {
			return errors.Errorf("container.Bindings[%d] inequal: %+v != %+v", n, b1, b2)
		}
	}
	return nil
}

func EventsEqual(a *beacon.Event, b *beacon.Event) error {
	if a.Action != b.Action {
		return errors.Errorf("event.Action inequal: %s != %s", a.Action, b.Action)
	}
	if err := ContainersEqual(a.Container, b.Container); err != nil {
		return errors.Wrap(err, "event.Container inequal")
	}
	return nil
}

func EventArraysEqual(a []*beacon.Event, b []*beacon.Event) error {
	if len(a) != len(b) {
		return errors.Errorf("event arrays have inequal length: %d != %d", len(a), len(b))
	}
	for n := range a {
		if err := EventsEqual(a[n], b[n]); err != nil {
			return errors.Wrapf(err, "events[%d] inequal", n)
		}
	}
	return nil
}

func testEvents(t *testing.T, events []*beacon.Event) {
	eventsChan := make(chan *beacon.Event, 1)
	server := NewServer(t, eventsChan)
	defer server.Close()
	backend := sns.NewWithEndpoint(server.URL, TEST_REGION, TEST_TOPIC)

	go func() {
		for _, event := range events {
			if err := backend.ProcessEvent(event); err != nil {
				t.Fatal(err)
			}
		}
	}()

	haveEvents, err := WaitForEvents(eventsChan, len(events), 5*time.Second)
	if err != nil {
		t.Error(err)
	} else if err := EventArraysEqual(haveEvents, events); err != nil {
		t.Error(err)
	}
}

func TestOneEvent(t *testing.T) {
	t.Parallel()
	testEvents(t, []*beacon.Event{
		{
			Action: beacon.Start,
			Container: &beacon.Container{
				ID:      RandomID(),
				Service: "test",
				Labels: map[string]string{
					"service": "test",
					"test":    "TestOneEvent",
				},
				Bindings: []*beacon.Binding{
					{HostIP: "0.0.0.0", HostPort: 54392, ContainerPort: 80, Protocol: beacon.TCP},
				},
			},
		},
	})
}

func TestTwoEvents(t *testing.T) {
	t.Parallel()
	testEvents(t, []*beacon.Event{
		{
			Action: beacon.Start,
			Container: &beacon.Container{
				ID:      RandomID(),
				Service: "test",
				Labels: map[string]string{
					"service": "test",
					"test":    "TestTwoEvent",
				},
				Bindings: []*beacon.Binding{
					{HostIP: "0.0.0.0", HostPort: 54392, ContainerPort: 80, Protocol: beacon.TCP},
				},
			},
		},
		{
			Action: beacon.Start,
			Container: &beacon.Container{
				ID:      RandomID(),
				Service: "test",
				Labels: map[string]string{
					"service": "test",
					"test":    "TestTestEvent",
				},
				Bindings: []*beacon.Binding{
					{HostIP: "0.0.0.0", HostPort: 54363, ContainerPort: 80, Protocol: beacon.TCP},
				},
			},
		},
	})
}
