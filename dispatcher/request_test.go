package dispatcher

import (
	"os/exec"
	"strings"
	"testing"
)

func TestGetLocalIP(t *testing.T) {
	out, err := exec.Command("hostname", "-I").CombinedOutput()
	if err != nil {
		t.Fatalf("exec `hostname -I`: %v", err)
	}
	sysIP := strings.TrimSpace(string(out))
	localIP := getLocalIP()
	if localIP != sysIP {
		t.Errorf("getLocalIP: expected %s, got %s", sysIP, localIP)
	}
}
