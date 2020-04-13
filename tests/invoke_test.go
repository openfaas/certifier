package tests

import (
	"net/http"
	"strings"
	"testing"

	"fmt"

	faasSDK "github.com/openfaas/faas-cli/proxy"
)

func Test_InvokeNotFound(t *testing.T) {
	_ = invoke(t, "notfound", "", http.StatusNotFound, http.StatusBadGateway)
}

func Test_Invoke_With_Supported_Verbs(t *testing.T) {
	envVars := map[string]string{}
	functionRequest := &faasSDK.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: "env-test-verbs",
		Network:      "func_functions",
		FProcess:     "env",
		EnvVars:      envVars,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
		return
	}

	list(t, http.StatusOK)

	verbs := []struct {
		verb  string
		match func(string) bool
	}{
		{verb: http.MethodGet, match: func(r string) bool { return strings.Contains(r, "Http_Method=GET") }},
		{verb: http.MethodPost, match: func(r string) bool { return strings.Contains(r, "Http_Method=POST") }},
		{verb: http.MethodPut, match: func(r string) bool { return strings.Contains(r, "Http_Method=PUT") }},
		{verb: http.MethodPatch, match: func(r string) bool { return strings.Contains(r, "Http_Method=PATCH") }},
		{verb: http.MethodDelete, match: func(r string) bool { return strings.Contains(r, "Http_Method=DELETE") }},
	}

	for _, v := range verbs {
		t.Run(v.verb, func(t *testing.T) {

			bytesOut := invokeWithVerb(t, v.verb, functionRequest.FunctionName, emptyQueryString, http.StatusOK)

			out := string(bytesOut)
			if !v.match(out) {
				t.Fatalf("want: %s, got: %s", fmt.Sprintf("Http_Method=%s", v.verb), out)
			}
		})
	}
}
