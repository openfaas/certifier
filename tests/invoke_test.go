package tests

import (
	"net/http"
	"strings"
	"testing"

	"fmt"

	sdk "github.com/openfaas/faas-cli/proxy"
	types "github.com/openfaas/faas-provider/types"
)

func Test_InvokeNotFound(t *testing.T) {
	functionRequest := &sdk.DeployFunctionSpec{
		Image:     "notfound",
		Namespace: config.DefaultNamespace,
	}
	_ = invoke(t, functionRequest, "", "", http.StatusNotFound, http.StatusBadGateway)
}

func invokeWithSupportedVerbs(t *testing.T, functionRequest *sdk.DeployFunctionSpec) {
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

			bytesOut, res := invokeWithVerb(t, v.verb, functionRequest, emptyQueryString, "", http.StatusOK)

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

func invokeWithCustomEnvVarsAndQueryString(t *testing.T, functionRequest *sdk.DeployFunctionSpec) {
	t.Run("Empty QueryString", func(t *testing.T) {
		bytesOut := invoke(t, functionRequest, emptyQueryString, "", http.StatusOK)
		out := string(bytesOut)
		if strings.Contains(out, "custom_env") == false {
			t.Fatalf("want: %s, got: %s", "custom_env", out)
		}
	})

	t.Run("Populated QueryString", func(t *testing.T) {
		bytesOut := invoke(t, functionRequest, "testing=1", "", http.StatusOK)
		out := string(bytesOut)
		if strings.Contains(out, "Http_Query=testing=1") == false {
			t.Fatalf("want: %s, got: %s", "Http_Query=testing=1", out)
		}
	})
}

func Test_Invoke(t *testing.T) {
	t.Logf("Gateway: %s", config.Gateway)
	imagePrefix := config.RegistryPrefix + "/"
	cases := []FunctionTestCase{
		{
			name: "Invoke test with different verbs",
			function: types.FunctionDeployment{
				Image:      imagePrefix + "functions/alpine:latest",
				Service:    "env-test-verbs",
				EnvProcess: "env",
				EnvVars:    map[string]string{},
				Namespace:  config.DefaultNamespace,
			},
		},
		{
			name: "Invoke propogates redirect to the caller",
			function: types.FunctionDeployment{
				Image:      imagePrefix + "theaxer/redirector:latest",
				Service:    "redirector-test",
				EnvProcess: "./handler",
				EnvVars:    map[string]string{"destination": "http://example.com"},
				Namespace:  config.DefaultNamespace,
			},
		},
		{
			name: "Invoke with custom env vars and query string",
			function: types.FunctionDeployment{
				Image:      imagePrefix + "functions/alpine:latest",
				Service:    "env-test",
				EnvProcess: "env",
				EnvVars:    map[string]string{"custom_env": "custom_env_value"},
				Namespace:  config.DefaultNamespace,
			},
		},
	}

	cases = copyNamespacesTest(cases)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			functionRequest := createDeploymentSpec(c)
			deployStatus := deploy(t, functionRequest)
			if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
				t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
				return
			}

			fns := list(t, http.StatusOK, functionRequest.Namespace)

			inList := false
			for _, fn := range fns {
				if fn.Name == functionRequest.FunctionName && fn.Namespace == functionRequest.Namespace {
					inList = true
					break
				}
			}
			if !inList {
				t.Fatalf("want function %s in list, but was not found", functionRequest.FunctionName)
			}

			status := get(t, functionRequest.FunctionName, functionRequest.Namespace)

			if status.Image != functionRequest.Image {
				t.Fatalf("function status image - want %s, but got %s",
					functionRequest.Image,
					status.Image)
			}

			switch service := c.function.Service; service {
			case "env-test-verbs":
				invokeWithSupportedVerbs(t, functionRequest)
			case "redirector-test":
				_ = invoke(t, functionRequest, emptyQueryString, "", http.StatusFound)
			case "env-test":
				invokeWithCustomEnvVarsAndQueryString(t, functionRequest)
			default:
				t.Fatalf("Invoke tests does not handle %s. Please raise an issue on repository", c.function.Service)
			}
		})
	}
}
