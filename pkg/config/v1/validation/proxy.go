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

package validation

import (
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func validateProxyBaseConfigForClient(c *v1.ProxyBaseConfig) error {
	if c.Name == "" {
		return errors.New("name should not be empty")
	}
	if err := ValidateAnnotations(c.Annotations); err != nil {
		return err
	}
	if err := ValidatePort(c.LocalPort, "localPort"); err != nil {
		return fmt.Errorf("localPort: %v", err)
	}
	return nil
}

func validateProxyBaseConfigForServer(c *v1.ProxyBaseConfig) error {
	if err := ValidateAnnotations(c.Annotations); err != nil {
		return err
	}
	return nil
}

func ValidateProxyConfigurerForClient(c v1.ProxyConfigurer) error {
	base := c.GetBaseConfig()
	if err := validateProxyBaseConfigForClient(base); err != nil {
		return err
	}
	switch c.(type) {
	case *v1.XTCPProxyConfig:
		return nil
	}
	return errors.New("unknown proxy config type")
}

func ValidateProxyConfigurerForServer(c v1.ProxyConfigurer, _ *v1.ServerConfig) error {
	base := c.GetBaseConfig()
	if err := validateProxyBaseConfigForServer(base); err != nil {
		return err
	}
	switch c.(type) {
	case *v1.XTCPProxyConfig:
		return nil
	default:
		return errors.New("unknown proxy config type")
	}
}

func ValidateAnnotations(annotations map[string]string) error {
	if len(annotations) == 0 {
		return nil
	}

	var errs error
	for k := range annotations {
		for _, msg := range validation.IsQualifiedName(strings.ToLower(k)) {
			errs = AppendError(errs, fmt.Errorf("annotation key %s is invalid: %s", k, msg))
		}
	}
	if err := ValidateAnnotationsSize(annotations); err != nil {
		errs = AppendError(errs, err)
	}
	return errs
}

const TotalAnnotationSizeLimitB int = 256 * (1 << 10)

func ValidateAnnotationsSize(annotations map[string]string) error {
	var totalSize int64
	for k, v := range annotations {
		totalSize += (int64)(len(k)) + (int64)(len(v))
	}
	if totalSize > (int64)(TotalAnnotationSizeLimitB) {
		return fmt.Errorf("annotations size %d is larger than limit %d", totalSize, TotalAnnotationSizeLimitB)
	}
	return nil
}
