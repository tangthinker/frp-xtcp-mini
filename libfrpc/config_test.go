// Copyright 2026 The frp Authors
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

package libfrpc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseJSONAllocatesVisitorPortAndDefaultsLoginFailExit(t *testing.T) {
	parsed, err := parseClientConfig(`{
		"serverAddr": "127.0.0.1",
		"serverPort": 7000,
		"auth": {"method": "token", "token": "x"},
		"visitors": [{
			"name": "ssh-visitor",
			"type": "xtcp",
			"serverName": "ssh",
			"bindPort": 0
		}]
	}`)
	require.NoError(t, err)
	require.NotNil(t, parsed.common.LoginFailExit)
	require.False(t, *parsed.common.LoginFailExit)
	require.Len(t, parsed.visitorsInfo, 1)
	require.Greater(t, parsed.visitorsInfo[0].BindPort, 0)
	require.True(t, strings.HasPrefix(parsed.visitorsInfo[0].URL, "http://127.0.0.1:"))
}

func TestParseTOML(t *testing.T) {
	parsed, err := parseClientConfig(`
serverAddr = "127.0.0.1"
serverPort = 7000
auth.method = "token"
auth.token = "x"

[[visitors]]
name = "ssh-visitor"
type = "xtcp"
serverName = "ssh"
bindPort = 6000
`)
	require.NoError(t, err)
	require.Equal(t, 6000, parsed.visitorsInfo[0].BindPort)
	require.Equal(t, "http://127.0.0.1:6000", parsed.visitorsInfo[0].URL)
}

func TestParseRespectsExplicitLoginFailExit(t *testing.T) {
	parsed, err := parseClientConfig(`{
		"serverAddr": "127.0.0.1",
		"loginFailExit": true,
		"visitors": [{
			"name": "v",
			"type": "xtcp",
			"serverName": "p",
			"bindPort": 1
		}]
	}`)
	require.NoError(t, err)
	require.True(t, *parsed.common.LoginFailExit)
}

func TestParseRejectsEmptyConfig(t *testing.T) {
	_, err := parseClientConfig("")
	require.Equal(t, CodeInvalidArg, CodeOf(err))
}

func TestParseRejectsInvalidJSON(t *testing.T) {
	_, err := parseClientConfig("{")
	require.Equal(t, CodeInvalidJSON, CodeOf(err))
}

func TestParseRejectsInvalidVisitor(t *testing.T) {
	_, err := parseClientConfig(`{
		"serverAddr": "127.0.0.1",
		"visitors": [{"name": "v", "type": "xtcp", "bindPort": 1}]
	}`)
	require.Equal(t, CodeInvalidConfig, CodeOf(err))
	require.Contains(t, err.Error(), "server name")
}

func TestLocalHTTPURLMapsWildcardToLoopback(t *testing.T) {
	require.Equal(t, "http://127.0.0.1:80", localHTTPURL("0.0.0.0", 80))
	require.Equal(t, "http://127.0.0.1:80", localHTTPURL("::", 80))
}

func TestErrorCodesMatchHeader(t *testing.T) {
	header, err := os.ReadFile(filepath.Join("frpc.h"))
	require.NoError(t, err)
	text := string(header)
	require.Contains(t, text, "#define FRPC_API_VERSION 1")
	require.Contains(t, text, "#define FRPC_OK 0")
	require.Contains(t, text, "#define FRPC_ERR_INVALID_ARG -1")
	require.Contains(t, text, "#define FRPC_ERR_INVALID_JSON -2")
	require.Contains(t, text, "#define FRPC_ERR_INVALID_CONFIG -3")
	require.Contains(t, text, "#define FRPC_ERR_NOT_FOUND -4")
	require.Contains(t, text, "#define FRPC_ERR_NOT_CONNECTED -5")
	require.Contains(t, text, "#define FRPC_ERR_TIMEOUT -6")
	require.Contains(t, text, "#define FRPC_ERR_STOPPED -7")
	require.Contains(t, text, "#define FRPC_ERR_FAILED -8")
	require.Contains(t, text, "#define FRPC_ERR_INTERNAL -9")
}

func TestCodeStringStable(t *testing.T) {
	require.Equal(t, "ok", CodeOK.String())
	require.Equal(t, "handle not found", CodeNotFound.String())
}

func TestStatusJSONRoundTripShape(t *testing.T) {
	st := Status{
		APIVersion:     APIVersion,
		Handle:         7,
		State:          StateConnected,
		RunID:          "abc",
		ReconnectCount: 2,
		ServerAddr:     "127.0.0.1",
		ServerPort:     7000,
		Visitors: []Endpoint{{
			Name: "ssh-visitor", Type: "xtcp", ServerName: "ssh",
			BindAddr: "127.0.0.1", BindPort: 6000, URL: "http://127.0.0.1:6000",
		}},
	}
	b, err := json.Marshal(st)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(b, &decoded))
	require.Equal(t, float64(1), decoded["apiVersion"])
	require.Equal(t, "connected", decoded["state"])
	require.Equal(t, float64(2), decoded["reconnectCount"])
}
