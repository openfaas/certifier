package tests

import (
	"net/http"
	"time"
)

type Unauthenticated struct {
}

func (auth *Unauthenticated) Set(req *http.Request) error {
	return nil
}

var (
	timeout          = 5 * time.Second
	defaultNamespace = ""
)
