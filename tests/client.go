package tests

import (
	"net/http"

	sdk "github.com/openfaas/faas-cli/proxy"
)

// Unauthenticated implements the sdk ClientAuthSetter as a noop, leaving
// the request unauthneticated
type Unauthenticated struct {
	sdk.ClientAuth
}

func (auth *Unauthenticated) Set(req *http.Request) error {
	return nil
}
