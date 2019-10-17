// Copyright 2019 Kosc Telecom.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
