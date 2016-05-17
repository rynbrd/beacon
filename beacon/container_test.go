package beacon_test

import (
	beacon "."
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
