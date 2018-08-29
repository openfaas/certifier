package tests

import (
	"net/http"
	"testing"
	"strings"

	"github.com/openfaas/faas/gateway/requests"
	"fmt"
)

func Test_InvokeNotFound(t *testing.T) {
	invoke(t, "notfound", emptyQueryString, http.StatusNotFound, http.StatusBadGateway)
}

func Test_Invoke_With_Supported_Verbs(t *testing.T) {
	envVars := map[string]string{}
	functionRequest := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    "env-test",
		Network:    "func_functions",
		EnvProcess: "env",
		EnvVars:    envVars,
	}

	deployStatus, deployErr := deploy(t, functionRequest)
	if deployErr != nil {
		t.Log(deployErr.Error())
		t.Fail()
	}

	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Logf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
		t.Fail()
	}

	list(t, http.StatusOK)

	verbs :=[]struct {
		verb        string
		expect      func(string) bool
	}{
		{ verb: "GET", expect: func(r string) bool { return  strings.Contains(r, "Http_Method=GET")}},
		{ verb: "POST", expect: func(r string) bool { return  strings.Contains(r, "Http_Method=POST")}},
		{ verb: "PUT", expect: func(r string) bool { return  strings.Contains(r, "Http_Method=PUT")}},
		{ verb: "PATCH", expect: func(r string) bool { return  strings.Contains(r, "Http_Method=PATCH")}},
		{ verb: "DELETE", expect: func(r string) bool { return  strings.Contains(r, "Http_Method=DELETE")}},
	}

	for _, v := range verbs {
		t.Run(v.verb, func(t *testing.T) {
			bytesOut := invokeWithVerb(t, v.verb, functionRequest.Service, emptyQueryString, http.StatusOK)

			out := string(bytesOut)
			if !v.expect(out) {
				t.Logf("want: %s, got: %s", fmt.Sprintf("Http_Method=%s", v.verb), out)
				t.Fail()
			}
		})
	}
}