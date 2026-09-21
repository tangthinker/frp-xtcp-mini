package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalTypedProxyConfig(t *testing.T) {
	require := require.New(t)
	proxyConfigs := struct {
		Proxies []TypedProxyConfig `json:"proxies,omitempty"`
	}{}

	strs := `{
		"proxies": [
			{
				"type": "xtcp",
				"name": "ssh",
				"localPort": 22,
				"secretKey": "abcdefg"
			}
		]
	}`
	err := json.Unmarshal([]byte(strs), &proxyConfigs)
	require.NoError(err)
	require.IsType(&XTCPProxyConfig{}, proxyConfigs.Proxies[0].ProxyConfigurer)
}
