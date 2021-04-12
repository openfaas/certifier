package tests

import (
	"net/http"
	"strings"
	"testing"

	"fmt"

	sdk "github.com/openfaas/faas-cli/proxy"
)

func Test_InvokeNotFound(t *testing.T) {
	_ = invoke(t, "notfound", "", http.StatusNotFound, http.StatusBadGateway)
}

func Test_Invoke_With_Supported_Verbs(t *testing.T) {
	envVars := map[string]string{}
	functionRequest := &sdk.DeployFunctionSpec{
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

			bytesOut, res := invokeWithVerb(t, v.verb, functionRequest.FunctionName, emptyQueryString, http.StatusOK)

			out := string(bytesOut)
			if !v.match(out) {
				t.Fatalf("want: %s, got: %s", fmt.Sprintf("Http_Method=%s", v.verb), out)
			}

			callID := res.Header.Get("X-Call-Id")
			if callID == "" {
				t.Fatal("expect non-empty X-Call-Id header")
			}

			startTime := res.Header.Get("X-Start-Time")
			if startTime == "" {
				t.Fatal("expect non-empty X-Start-Time header")
			}
		})
	}
}

func Test_InvokePropogatesRedirectToTheCaller(t *testing.T) {
	destination := "http://example.com"
	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "theaxer/redirector:latest",
		FunctionName: "redirector-test",
		Network:      "func_functions",
		FProcess:     "./handler",
		EnvVars:      map[string]string{"destination": destination},
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
		return
	}

	_ = invoke(t, "redirector-test", emptyQueryString, http.StatusFound)
}

func Test_Invoke_With_CustomEnvVars_AndQueryString(t *testing.T) {
	envVars := map[string]string{}
	envVars["custom_env"] = "custom_env_value"

	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: "env-test",
		Network:      "func_functions",
		FProcess:     "env",
		EnvVars:      envVars,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	list(t, http.StatusOK)

	t.Run("Empty QueryString", func(t *testing.T) {
		bytesOut := invoke(t, functionRequest.FunctionName, emptyQueryString, http.StatusOK)
		out := string(bytesOut)
		if strings.Contains(out, "custom_env") == false {
			t.Fatalf("want: %s, got: %s", "custom_env", out)
		}
	})

	t.Run("Populated QueryString", func(t *testing.T) {
		bytesOut := invoke(t, functionRequest.FunctionName, "testing=1", http.StatusOK)
		out := string(bytesOut)
		if strings.Contains(out, "Http_Query=testing=1") == false {
			t.Fatalf("want: %s, got: %s", "Http_Query=testing=1", out)
		}
	})
}
