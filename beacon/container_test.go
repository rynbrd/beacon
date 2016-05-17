package beacon_test

import (
	beacon "."
	"reflect"
	"testing"
)

func TestContainerEqual(t *testing.T) {
	type Test struct {
		A     *beacon.Container
		B     *beacon.Container
		Equal bool
	}

	tests := []Test{
		{
			A: &beacon.Container{
				ID:      "1234",
				Service: "www",
				Labels: map[string]string{
					"a": "aye",
				},
				Bindings: []*beacon.Binding{
					{
						HostIP:        "127.0.0.1",
						HostPort:      54926,
						ContainerPort: 80,
						Protocol:      beacon.TCP,
					},
				},
			},
			B: &beacon.Container{
				ID:      "1234",
				Service: "www",
				Labels: map[string]string{
					"a": "aye",
				},
				Bindings: []*beacon.Binding{
					{
						HostIP:        "127.0.0.1",
						HostPort:      54926,
						ContainerPort: 80,
						Protocol:      beacon.TCP,
					},
				},
			},
			Equal: true,
		},
		{
			A:     nil,
			B:     nil,
			Equal: true,
		},
		{
			A: &beacon.Container{
				ID:      "1234",
				Service: "www",
				Labels: map[string]string{
					"a": "aye",
				},
				Bindings: []*beacon.Binding{
					{
						HostIP:        "127.0.0.1",
						HostPort:      54926,
						ContainerPort: 80,
						Protocol:      beacon.TCP,
					},
				},
			},
			B: &beacon.Container{
				ID:      "1234",
				Service: "www",
				Labels: map[string]string{
					"a": "eh",
				},
				Bindings: []*beacon.Binding{
					{
						HostIP:        "127.0.0.1",
						HostPort:      54926,
						ContainerPort: 80,
						Protocol:      beacon.TCP,
					},
				},
			},
			Equal: false,
		},
		{
			A: &beacon.Container{
				ID:      "1234",
				Service: "www",
				Labels: map[string]string{
					"a": "aye",
				},
				Bindings: []*beacon.Binding{
					{
						HostIP:        "127.0.0.1",
						HostPort:      54926,
						ContainerPort: 80,
						Protocol:      beacon.TCP,
					},
				},
			},
			B:     nil,
			Equal: false,
		},
		{
			A: &beacon.Container{
				ID:      "1234",
				Service: "www",
				Labels: map[string]string{
					"a": "aye",
				},
				Bindings: []*beacon.Binding{
					{
						HostIP:        "127.0.0.1",
						HostPort:      54926,
						ContainerPort: 80,
						Protocol:      beacon.TCP,
					},
				},
			},
			B: &beacon.Container{
				ID:      "1234",
				Service: "www",
				Labels: map[string]string{
					"a": "aye",
				},
				Bindings: []*beacon.Binding{
					{
						HostIP:        "127.0.0.1",
						HostPort:      54926,
						ContainerPort: 80,
						Protocol:      beacon.TCP,
					},
					{
						HostIP:        "127.0.0.1",
						HostPort:      54927,
						ContainerPort: 443,
						Protocol:      beacon.TCP,
					},
				},
			},
			Equal: false,
		},
		{
			A: &beacon.Container{
				ID:      "1234",
				Service: "www",
				Labels: map[string]string{
					"a": "aye",
				},
				Bindings: []*beacon.Binding{
					{
						HostIP:        "127.0.0.1",
						HostPort:      54926,
						ContainerPort: 80,
						Protocol:      beacon.TCP,
					},
				},
			},
			B: &beacon.Container{
				ID:      "1234",
				Service: "www",
				Labels: map[string]string{
					"a": "aye",
				},
				Bindings: []*beacon.Binding{
					{
						HostIP:        "127.0.0.1",
						HostPort:      54927,
						ContainerPort: 80,
						Protocol:      beacon.TCP,
					},
				},
			},
			Equal: false,
		},
	}

	for n, test := range tests {
		if test.A.Equal(test.B) != test.Equal {
			t.Errorf("containers %d inequal", n)
		}
	}
}

func TestContainerCopy(t *testing.T) {
	container := &beacon.Container{
		ID:      "123456",
		Service: "example",
		Labels: map[string]string{
			"a": "aye",
			"b": "bee",
		},
		Bindings: []*beacon.Binding{
			{
				HostIP:        "127.0.0.1",
				HostPort:      56291,
				ContainerPort: 80,
				Protocol:      beacon.TCP,
			},
			{
				HostIP:        "127.0.0.1",
				HostPort:      56292,
				ContainerPort: 443,
				Protocol:      beacon.TCP,
			},
		},
	}
	newContainer := container.Copy()

	if !reflect.DeepEqual(container, newContainer) {
		t.Errorf("container copy differs from original: %+v != %+v", container, newContainer)
	}

	newService := "www"
	container.Service = newService
	if newContainer.Service == newService {
		t.Error("container copy points to same memory space")
	}

	newLabel := "eh"
	container.Labels["a"] = newLabel
	if reflect.DeepEqual(container.Labels, newContainer.Labels) {
		t.Error("container.Labels copy points to same memory space")
	}

	newPort := 54293
	container.Bindings[1].HostPort = newPort
	if reflect.DeepEqual(container.Bindings, newContainer.Bindings) {
		t.Error("container.Bindings copy points to same memory space")
	}
}
