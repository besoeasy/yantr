package main

import (
	"testing"

	dockertypes "github.com/docker/docker/api/types"
	dockerctr "github.com/docker/docker/api/types/container"
	dockernat "github.com/docker/go-connections/nat"
)

func TestBrowserContainerName(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"myvol", "y-fs-myvol"},
		{"my vol!", "y-fs-my-vol-"},
		{"a/b\\c:d", "y-fs-a-b-c-d"},
	} {
		if got := browserContainerName(tc.in); got != tc.want {
			t.Errorf("browserContainerName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBrowserHostPort(t *testing.T) {
	bindings := func(port string) dockernat.PortMap {
		return dockernat.PortMap{
			"5000/tcp": {{HostIP: "127.0.0.1", HostPort: port}},
		}
	}
	mkInfo := func(netPorts, hostPorts dockernat.PortMap) dockertypes.ContainerJSON {
		ns := &dockerctr.NetworkSettings{}
		ns.Ports = netPorts
		return dockertypes.ContainerJSON{
			ContainerJSONBase: &dockerctr.ContainerJSONBase{
				HostConfig: &dockerctr.HostConfig{PortBindings: hostPorts},
			},
			NetworkSettings: ns,
		}
	}

	for _, tc := range []struct {
		name string
		info dockertypes.ContainerJSON
		port int
		ok   bool
	}{
		{"network settings", mkInfo(bindings("41234"), nil), 41234, true},
		{"hostconfig fallback", mkInfo(nil, bindings("41235")), 41235, true},
		{"no bindings", mkInfo(nil, nil), 0, false},
		{"non-numeric port", mkInfo(bindings("abc"), nil), 0, false},
		{"zero port", mkInfo(bindings("0"), nil), 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := browserHostPort(tc.info)
			if got != tc.port || ok != tc.ok {
				t.Errorf("browserHostPort() = (%d, %v), want (%d, %v)", got, ok, tc.port, tc.ok)
			}
		})
	}
}

func TestIsBrowserContainer(t *testing.T) {
	if !isBrowserContainer([]string{"/y-fs-data"}, nil) {
		t.Error("expected y-fs- name prefix to match")
	}
	if !isBrowserContainer([]string{"/other"}, map[string]string{"yantr.system": "browser"}) {
		t.Error("expected yantr.system=browser label to match")
	}
	if isBrowserContainer([]string{"/other"}, map[string]string{"yantr.system": "other"}) {
		t.Error("expected non-browser container to not match")
	}
}
