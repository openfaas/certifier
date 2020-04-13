package tests

import (
	"net/http"
	"time"
)

type FaaSAuth struct {
}

func (auth *FaaSAuth) Set(req *http.Request) error {
	return nil
}

var (
	timeout          = 5 * time.Second
	defaultNamespace = ""
)
