package langserver

import "github.com/sourcegraph/jsonrpc2"

func newJsonrpc2Errorf(code int64, message string) error {
	return &jsonrpc2.Error{Code: code, Message: message}
}
