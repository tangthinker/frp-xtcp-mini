module github.com/fatedier/frp

go 1.25.0

require (
	github.com/fatedier/golib v0.8.2
	github.com/google/uuid v1.6.0
	github.com/hashicorp/yamux v0.1.1
	github.com/onsi/ginkgo/v2 v2.23.4
	github.com/onsi/gomega v1.36.3
	github.com/pelletier/go-toml/v2 v2.2.0
	github.com/quic-go/quic-go v0.60.0
	github.com/samber/lo v1.47.0
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.11.1
	github.com/xtaci/kcp-go/v5 v5.6.13
	golang.org/x/net v0.56.0
	golang.org/x/sync v0.22.0
	golang.org/x/sys v0.47.0
	golang.org/x/time v0.10.0
	k8s.io/apimachinery v0.28.8
	k8s.io/utils v0.0.0-20230406110748-d93618cff8a2
)

require (
	github.com/Azure/go-ntlmssp v0.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/klauspost/reedsolomon v1.12.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/templexxx/cpu v0.1.1 // indirect
	github.com/templexxx/xorsimd v0.4.3 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/tools v0.47.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

// TODO(fatedier): Temporary use the modified version, update to the official version after merging into the official repository.
replace github.com/hashicorp/yamux => github.com/fatedier/yamux v0.0.0-20250825093530-d0154be01cd6
