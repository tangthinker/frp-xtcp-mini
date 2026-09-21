// Copyright 2023 The frp Authors
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

package config

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
)

func WordSepNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	if strings.Contains(name, "_") {
		return pflag.NormalizedName(strings.ReplaceAll(name, "_", "-"))
	}
	return pflag.NormalizedName(name)
}

func RegisterProxyFlags(cmd *cobra.Command, c v1.ProxyConfigurer) {
	registerProxyBaseConfigFlags(cmd, c.GetBaseConfig())

	if cc, ok := c.(*v1.XTCPProxyConfig); ok {
		cmd.Flags().StringVarP(&cc.Secretkey, "sk", "", "", "secret key")
		cmd.Flags().StringSliceVarP(&cc.AllowUsers, "allow_users", "", []string{}, "allow visitor users")
	}
}

func registerProxyBaseConfigFlags(cmd *cobra.Command, c *v1.ProxyBaseConfig) {
	if c == nil {
		return
	}
	cmd.Flags().StringVarP(&c.Name, "proxy_name", "n", "", "proxy name")
	cmd.Flags().StringToStringVarP(&c.Metadatas, "metadatas", "", nil, "metadata key-value pairs (e.g., key1=value1,key2=value2)")
	cmd.Flags().StringToStringVarP(&c.Annotations, "annotations", "", nil, "annotation key-value pairs (e.g., key1=value1,key2=value2)")
	cmd.Flags().StringVarP(&c.LocalIP, "local_ip", "i", "127.0.0.1", "local ip")
	cmd.Flags().IntVarP(&c.LocalPort, "local_port", "l", 0, "local port")
	cmd.Flags().BoolVarP(&c.Transport.UseEncryption, "ue", "", false, "use encryption")
	cmd.Flags().BoolVarP(&c.Transport.UseCompression, "uc", "", false, "use compression")
}

func RegisterVisitorFlags(cmd *cobra.Command, c v1.VisitorConfigurer) {
	registerVisitorBaseConfigFlags(cmd, c.GetBaseConfig())
}

func registerVisitorBaseConfigFlags(cmd *cobra.Command, c *v1.VisitorBaseConfig) {
	if c == nil {
		return
	}
	cmd.Flags().StringVarP(&c.Name, "visitor_name", "n", "", "visitor name")
	cmd.Flags().BoolVarP(&c.Transport.UseEncryption, "ue", "", false, "use encryption")
	cmd.Flags().BoolVarP(&c.Transport.UseCompression, "uc", "", false, "use compression")
	cmd.Flags().StringVarP(&c.SecretKey, "sk", "", "", "secret key")
	cmd.Flags().StringVarP(&c.ServerName, "server_name", "", "", "server name")
	cmd.Flags().StringVarP(&c.ServerUser, "server-user", "", "", "server user")
	cmd.Flags().StringVarP(&c.BindAddr, "bind_addr", "", "", "bind addr")
	cmd.Flags().IntVarP(&c.BindPort, "bind_port", "", 0, "bind port")
}

func RegisterClientCommonConfigFlags(cmd *cobra.Command, c *v1.ClientCommonConfig) {
	cmd.PersistentFlags().StringVarP(&c.ServerAddr, "server_addr", "s", "127.0.0.1", "frp server's address")
	cmd.PersistentFlags().IntVarP(&c.ServerPort, "server_port", "P", 7000, "frp server's port")
	cmd.PersistentFlags().StringVarP(&c.Transport.Protocol, "protocol", "p", "tcp",
		"optional values are "+strings.Join(validation.SupportedTransportProtocols, ", "))
	cmd.PersistentFlags().StringVarP(&c.Log.Level, "log_level", "", "info", "log level")
	cmd.PersistentFlags().StringVarP(&c.Log.To, "log_file", "", "console", "console or file path")
	cmd.PersistentFlags().Int64VarP(&c.Log.MaxDays, "log_max_days", "", 3, "log file reversed days")
	cmd.PersistentFlags().BoolVarP(&c.Log.DisablePrintColor, "disable_log_color", "", false, "disable log color in console")
	cmd.PersistentFlags().StringVarP(&c.Transport.TLS.ServerName, "tls_server_name", "", "", "specify the custom server name of tls certificate")
	cmd.PersistentFlags().StringVarP(&c.DNSServer, "dns_server", "", "", "specify dns server instead of using system default one")
	c.Transport.TLS.Enable = cmd.PersistentFlags().BoolP("tls_enable", "", true, "enable frpc tls")
	cmd.PersistentFlags().StringVarP(&c.User, "user", "u", "", "user")
	cmd.PersistentFlags().StringVar(&c.ClientID, "client-id", "", "unique identifier for this frpc instance")
	cmd.PersistentFlags().StringVarP(&c.Auth.Token, "token", "t", "", "auth token")
}

func RegisterServerConfigFlags(cmd *cobra.Command, c *v1.ServerConfig) {
	cmd.PersistentFlags().StringVarP(&c.BindAddr, "bind_addr", "", "0.0.0.0", "bind address")
	cmd.PersistentFlags().IntVarP(&c.BindPort, "bind_port", "p", 7000, "bind port")
	cmd.PersistentFlags().StringVarP(&c.Log.To, "log_file", "", "console", "log file")
	cmd.PersistentFlags().StringVarP(&c.Log.Level, "log_level", "", "info", "log level")
	cmd.PersistentFlags().Int64VarP(&c.Log.MaxDays, "log_max_days", "", 3, "log max days")
	cmd.PersistentFlags().BoolVarP(&c.Log.DisablePrintColor, "disable_log_color", "", false, "disable log color in console")
	cmd.PersistentFlags().StringVarP(&c.Auth.Token, "token", "t", "", "auth token")
	cmd.PersistentFlags().BoolVarP(&c.Transport.TLS.Force, "tls_only", "", false, "frps tls only")
}
