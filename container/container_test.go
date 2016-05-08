package container

import (
	"testing"
)

func TestContainerEnv(t *testing.T) {
	type testInput struct {
		env  []string
		name string
		want string
	}

	testInputs := []testInput{
		{[]string{}, "NOPE", ""},
		{[]string{"A=aye"}, "A", "aye"},
		{[]string{"A=aye", "B=bee", "C=see"}, "A", "aye"},
		{[]string{"A=aye", "B=bee", "C=see"}, "B", "bee"},
		{[]string{"A=aye", "B=bee", "C=see"}, "C", "see"},
		{[]string{"A=aye", "B=bee", "C=see"}, "NOPE", ""},
		{[]string{"A=aye", "A=eh", "B=bee", "C=sea", "C=see"}, "A", "aye"},
		{[]string{"A=aye", "A=eh", "B=bee", "C=sea", "C=see"}, "B", "bee"},
		{[]string{"A=aye", "A=eh", "B=bee", "C=sea", "C=see"}, "C", "sea"},
		{[]string{"A=aye", "A=eh", "B=bee", "C=sea", "C=see"}, "NOPE", ""},
	}

	for _, in := range testInputs {
		container := &Container{"example", in.env, "localhost", []*Mapping{}}
		have := container.Env(in.name)
		if have != in.want {
			t.Errorf("env %s value '%s' != '%s'", in.name, have, in.want)
		}
	}
}

func TestContainerMapping(t *testing.T) {
	type testInput struct {
		mappings [][2]string
		port     string
		want     string
	}

	containerIP := "172.16.0.32"
	testInputs := []testInput{
		{[][2]string{}, "80/tcp", ""},
		{[][2]string{{"10.1.1.1:49000/tcp", "80/tcp"}}, "80/udp", ""},
		{[][2]string{{"10.1.1.1:49000/tcp", "80/tcp"}}, "1643/udp", ""},
		{[][2]string{{"10.1.1.1:49000/tcp", "80/tcp"}}, "80/tcp", "10.1.1.1:49000/tcp"},
		{[][2]string{{"10.1.1.1:49000/tcp", "80/tcp"}, {"10.1.1.1:49001/tcp", "80/tcp"}}, "80/tcp", "10.1.1.1:49000/tcp"},
		{[][2]string{{"10.1.1.1:49000/tcp", "80/tcp"}, {"10.1.1.1:49001/udp", "1643/udp"}}, "1643/tcp", ""},
		{[][2]string{{"10.1.1.1:49000/tcp", "80/tcp"}, {"10.1.1.1:49001/udp", "1643/udp"}}, "1643/udp", "10.1.1.1:49001/udp"},
	}

	for _, in := range testInputs {
		port, _ := ParsePort(in.port)
		t.Logf("port: %+v", port)
		t.Log("mappings:")
		mappings := make([]*Mapping, len(in.mappings))
		for n, mappingStrs := range in.mappings {
			addr, _ := ParseAddress(mappingStrs[0])
			port, _ := ParsePort(mappingStrs[1])
			mappings[n] = &Mapping{addr, port}
			t.Logf("  %+v", mappings[n])
		}
		container := &Container{"example", []string{}, containerIP, mappings}

		var wantAddr *Address
		if in.want != "" {
			wantAddr, _ = ParseAddress(in.want)
		}

		haveAddr, err := container.Mapping(port)
		if in.want == "" && err != PortNotMapped {
			t.Errorf("port %s error '%s' != '%s'", in.port, err, PortNotMapped)
		} else if in.want != "" {
			if err != nil {
				t.Errorf("port %s error %s", in.port, err)
			} else if !wantAddr.Equal(haveAddr) {
				t.Errorf("port %s mapping %+v != %+v", in.port, haveAddr, wantAddr)
			}
		}
	}
}

func TestParseService(t *testing.T) {
	type testInput struct {
		str  string
		want *Service
	}

	testInputs := []testInput{
		{"", nil},
		{"example", nil},
		{"example:80", &Service{"example", &Port{80, "tcp"}}},
		{"example:80/tcp", &Service{"example", &Port{80, "tcp"}}},
		{" example:80/tcp", &Service{"example", &Port{80, "tcp"}}},
		{"   example:80/tcp", &Service{"example", &Port{80, "tcp"}}},
		{"example:80/tcp ", &Service{"example", &Port{80, "tcp"}}},
		{"example:80/tcp   ", &Service{"example", &Port{80, "tcp"}}},
		{" example:80/tcp ", &Service{"example", &Port{80, "tcp"}}},
		{"   example:80/tcp  ", &Service{"example", &Port{80, "tcp"}}},
		{"example:1643/udp", &Service{"example", &Port{1643, "udp"}}},
		{":1643/udp", nil},
		{":", nil},
	}

	for _, in := range testInputs {
		svc, err := ParseService(in.str)
		if in.want == nil && err == nil {
			t.Errorf("no error for %s", in.str)
		} else if in.want != nil {
			if err != nil {
				t.Errorf("error for %s: %s", in.str, err)
			} else if in.want.Name != svc.Name || !in.want.Port.Equal(svc.Port) {
				t.Errorf("svc %+v != %+v", svc, in.want)
			}
		}
	}
}
