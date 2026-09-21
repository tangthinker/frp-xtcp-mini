package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeProxyConfigurerJSON_UnknownType(t *testing.T) {
	require := require.New(t)

	_, err := DecodeProxyConfigurerJSON([]byte(`{"name":"p1","type":"tcp"}`), DecodeOptions{})
	require.ErrorContains(err, "unknown proxy type")
}

func TestDecodeVisitorConfigurerJSON_UnknownType(t *testing.T) {
	require := require.New(t)

	_, err := DecodeVisitorConfigurerJSON([]byte(`{"name":"v1","type":"stcp"}`), DecodeOptions{})
	require.ErrorContains(err, "unknown visitor type")
}

func TestDecodeClientConfigJSON_StrictUnknownProxyField(t *testing.T) {
	require := require.New(t)

	data := []byte(`{
		"serverPort":7000,
		"proxies":[
			{
				"name":"p1",
				"type":"xtcp",
				"localPort":10080,
				"unknownField":"value"
			}
		]
	}`)

	_, err := DecodeClientConfigJSON(data, DecodeOptions{DisallowUnknownFields: false})
	require.NoError(err)

	_, err = DecodeClientConfigJSON(data, DecodeOptions{DisallowUnknownFields: true})
	require.ErrorContains(err, "unknownField")
}
