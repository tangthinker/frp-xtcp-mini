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
	"errors"
	"fmt"
)

// Code is a stable FFI error code. Values must stay in sync with frpc.h.
type Code int32

const (
	CodeOK            Code = 0
	CodeInvalidArg    Code = -1
	CodeInvalidJSON   Code = -2
	CodeInvalidConfig Code = -3
	CodeNotFound      Code = -4
	CodeNotConnected  Code = -5
	CodeTimeout       Code = -6
	CodeStopped       Code = -7
	CodeFailed        Code = -8
	CodeInternal      Code = -9
)

// APIVersion is bumped only when the JSON protocol is not backward compatible.
const APIVersion = 1

// Error is a coded error returned by the libfrpc API.
type Error struct {
	Code Code
	Msg  string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Msg == "" {
		return e.Code.String()
	}
	return e.Msg
}

func (c Code) String() string {
	switch c {
	case CodeOK:
		return "ok"
	case CodeInvalidArg:
		return "invalid argument"
	case CodeInvalidJSON:
		return "invalid config json"
	case CodeInvalidConfig:
		return "invalid config"
	case CodeNotFound:
		return "handle not found"
	case CodeNotConnected:
		return "not connected"
	case CodeTimeout:
		return "timeout"
	case CodeStopped:
		return "session stopped"
	case CodeFailed:
		return "session failed"
	case CodeInternal:
		return "internal error"
	default:
		return fmt.Sprintf("error %d", int32(c))
	}
}

func wrap(code Code, err error) error {
	if err == nil {
		return nil
	}
	var coded *Error
	if errors.As(err, &coded) {
		return coded
	}
	return &Error{Code: code, Msg: err.Error()}
}

func newError(code Code, msg string) error {
	return &Error{Code: code, Msg: msg}
}

// CodeOf maps an error to a stable FFI code.
func CodeOf(err error) Code {
	if err == nil {
		return CodeOK
	}
	var coded *Error
	if errors.As(err, &coded) {
		return coded.Code
	}
	return CodeInternal
}
