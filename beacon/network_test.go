package beacon_test

import (
	beacon "."
	"testing"
)

func TestBindingEqual(t *testing.T) {
	type Test struct {
		A     *beacon.Binding
		B     *beacon.Binding
		Equal bool
	}

	tests := []Test{
		{
			A: &beacon.Binding{
				HostIP:        "127.0.0.1",
				HostPort:      54926,
				ContainerPort: 80,
				Protocol:      beacon.TCP,
			},
			B: &beacon.Binding{
				HostIP:        "127.0.0.1",
				HostPort:      54926,
				ContainerPort: 80,
				Protocol:      beacon.TCP,
			},
			Equal: true,
		},
		{
			A: &beacon.Binding{
				HostIP:        "127.0.0.1",
				HostPort:      54926,
				ContainerPort: 80,
				Protocol:      beacon.TCP,
			},
			B: &beacon.Binding{
				HostIP:        "127.0.0.1",
				HostPort:      54927,
				ContainerPort: 80,
				Protocol:      beacon.TCP,
			},
			Equal: false,
		},
		{
			A: &beacon.Binding{
				HostIP:        "127.0.0.1",
				HostPort:      54926,
				ContainerPort: 80,
				Protocol:      beacon.TCP,
			},
			B: &beacon.Binding{
				HostIP:        "10.1.1.12",
				HostPort:      54926,
				ContainerPort: 80,
				Protocol:      beacon.TCP,
			},
			Equal: false,
		},
		{
			A: &beacon.Binding{
				HostIP:        "127.0.0.1",
				HostPort:      54926,
				ContainerPort: 80,
				Protocol:      beacon.TCP,
			},
			B: &beacon.Binding{
				HostIP:        "127.0.0.1",
				HostPort:      54926,
				ContainerPort: 443,
				Protocol:      beacon.TCP,
			},
			Equal: false,
		},
		{
			A: &beacon.Binding{
				HostIP:        "127.0.0.1",
				HostPort:      54926,
				ContainerPort: 80,
				Protocol:      beacon.TCP,
			},
			B: &beacon.Binding{
				HostIP:        "127.0.0.1",
				HostPort:      54926,
				ContainerPort: 80,
				Protocol:      beacon.UDP,
			},
			Equal: false,
		},
	}

	for n, test := range tests {
		if test.A.Equal(test.B) != test.Equal {
			t.Errorf("binding %d inequal", n)
		}
	}
}
