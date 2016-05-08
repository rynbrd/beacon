package container

import (
	"testing"
)

func TestParsePort(t *testing.T) {
	type testInput struct {
		str  string
		port *Port
	}

	testInputs := []testInput{
		{"80", &Port{80, "tcp"}},
		{"80/tcp", &Port{80, "tcp"}},
		{"1645/udp", &Port{1645, "udp"}},
		{" 80/tcp", &Port{80, "tcp"}},
		{"   80/tcp", &Port{80, "tcp"}},
		{"   80/tcp ", &Port{80, "tcp"}},
		{"   80/tcp   ", &Port{80, "tcp"}},
		{"80/tcp ", &Port{80, "tcp"}},
		{"80/tcp   ", &Port{80, "tcp"}},
		{"80 /tcp   ", nil},
		{"80 / tcp   ", nil},
		{"80/ tcp   ", nil},
		{"80.3/tcp", nil},
		{"80/icmp", nil},
		{"not a port", nil},
	}

	for _, in := range testInputs {
		port, err := ParsePort(in.str)
		if in.port == nil && err == nil {
			t.Errorf("no error for %s", in.str)
		} else if in.port != nil {
			if err != nil {
				t.Errorf("port %+v error %s", in.port, err)
			} else if in.port.Number != port.Number && in.port.Protocol != port.Protocol {
				t.Errorf("incorrect port %+v != %+v", port, in.port)
			}
		}
	}
}

func TestPortEqual(t *testing.T) {
	type testInput struct {
		left  *Port
		right *Port
		want  bool
	}

	testInputs := []testInput{
		{&Port{80, "tcp"}, &Port{80, "tcp"}, true},
		{&Port{1645, "udp"}, &Port{1645, "udp"}, true},
		{&Port{1645, "udp"}, &Port{80, "tcp"}, false},
		{&Port{1645, "udp"}, nil, false},
	}

	for _, in := range testInputs {
		if in.left.Equal(in.right) != in.want {
			if in.want {
				t.Errorf("port %+v != %+v", in.left, in.right)
			} else {
				t.Errorf("port %+v == %+v", in.left, in.right)
			}
		}
	}
}

func TestPortString(t *testing.T) {
	type testInput struct {
		port *Port
		want string
	}

	testInputs := []testInput{
		{&Port{80, "tcp"}, "80/tcp"},
		{&Port{1645, "udp"}, "1645/udp"},
	}

	for _, in := range testInputs {
		have := in.port.String()
		if have != in.want {
			t.Errorf("'%s' != '%s'", have, in.want)
		}
	}
}

func TestParseAddress(t *testing.T) {
	type testInput struct {
		str  string
		addr *Address
	}

	testInputs := []testInput{
		{"localhost:80/tcp", &Address{"localhost", &Port{80, "tcp"}}},
		{"localhost:80", &Address{"localhost", &Port{80, "tcp"}}},
		{"localhost:1645/udp", &Address{"localhost", &Port{1645, "udp"}}},
		{"localhost", nil},
		{"", nil},
		{":", nil},
		{":80/tcp", &Address{"0.0.0.0", &Port{80, "tcp"}}},
		{":80", &Address{"0.0.0.0", &Port{80, "tcp"}}},
	}

	for _, in := range testInputs {
		addr, err := ParseAddress(in.str)
		if in.addr == nil && err == nil {
			t.Errorf("no error for %s", in.str)
		} else if in.addr != nil {
			if in.addr.Hostname != addr.Hostname || !in.addr.Port.Equal(addr.Port) {
				t.Errorf("address %+v != %+v", addr, in.addr)
			} else if err != nil {
				t.Errorf("error for %s: %s", in.str, err)
			}
		}
	}
}

func TestAddressEqual(t *testing.T) {
	type testInput struct {
		left  *Address
		right *Address
		want  bool
	}

	testInputs := []testInput{
		{&Address{"localhost", &Port{80, "tcp"}}, &Address{"localhost", &Port{80, "tcp"}}, true},
		{&Address{"localhost", &Port{80, "tcp"}}, &Address{"0.0.0.0", &Port{80, "tcp"}}, false},
		{&Address{"localhost", &Port{80, "tcp"}}, &Address{"127.0.0.1", &Port{80, "tcp"}}, false},
		{&Address{"localhost", &Port{80, "tcp"}}, nil, false},
		{&Address{"localhost", &Port{1645, "udp"}}, &Address{"localhost", &Port{80, "tcp"}}, false},
		{&Address{"localhost", &Port{1645, "udp"}}, &Address{"localhost", &Port{1645, "udp"}}, true},
	}

	for _, in := range testInputs {
		if in.left.Equal(in.right) != in.want {
			if in.want {
				t.Errorf("address %+v != %+v", in.left, in.right)
			} else {
				t.Errorf("address %+v == %+v", in.left, in.right)
			}
		}
	}
}

func TestAddressString(t *testing.T) {
	type testInput struct {
		addr *Address
		want string
	}

	testInputs := []testInput{
		{&Address{"localhost", &Port{80, "tcp"}}, "localhost:80/tcp"},
		{&Address{"localhost", &Port{1645, "udp"}}, "localhost:1645/udp"},
		{&Address{"0.0.0.0", &Port{80, "tcp"}}, "0.0.0.0:80/tcp"},
		{&Address{"127.0.0.1", &Port{80, "tcp"}}, "127.0.0.1:80/tcp"},
	}

	for _, in := range testInputs {
		have := in.addr.String()
		if have != in.want {
			t.Errorf("'%s' != '%s'", have, in.want)
		}
	}
}

func TestAddressStringNoProtocol(t *testing.T) {
	type testInput struct {
		addr *Address
		want string
	}

	testInputs := []testInput{
		{&Address{"localhost", &Port{80, "tcp"}}, "localhost:80"},
		{&Address{"localhost", &Port{1645, "udp"}}, "localhost:1645"},
		{&Address{"0.0.0.0", &Port{80, "tcp"}}, "0.0.0.0:80"},
		{&Address{"127.0.0.1", &Port{80, "tcp"}}, "127.0.0.1:80"},
	}

	for _, in := range testInputs {
		have := in.addr.StringNoProtocol()
		if have != in.want {
			t.Errorf("'%s' != '%s'", have, in.want)
		}
	}
}
