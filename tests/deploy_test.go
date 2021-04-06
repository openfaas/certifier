package tests

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"

	sdk "github.com/openfaas/faas-cli/proxy"
)

var emptyQueryString = ""

func Test_Deploy_Stronghash(t *testing.T) {
	envVars := map[string]string{}
	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: "stronghash",
		Network:      "func_functions",
		FProcess:     "sha512sum",
		EnvVars:      envVars,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	list(t, http.StatusOK)
}

func Test_Deploy_PassingCustomEnvVars_AndQueryString(t *testing.T) {
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

func Test_Deploy_WithLabels(t *testing.T) {
	wantedLabels := map[string]string{
		"upstream_uri": "example.com",
		"canary_build": "true",
	}
	envVars := map[string]string{}

	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: "env-test-labels",
		Network:      "func_functions",
		FProcess:     "env",
		Labels:       wantedLabels,
		EnvVars:      envVars,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	_ = invoke(t, functionRequest.FunctionName, emptyQueryString, http.StatusOK)
	function := get(t, functionRequest.FunctionName)
	if err := strMapEqual("labels", *function.Labels, wantedLabels); err != nil {
		t.Fatal(err)
	}
}

func Test_Deploy_WithAnnotations(t *testing.T) {
	wantedAnnotations := map[string]string{
		"important-date": "Fri Aug 10 08:21:00 BST 2018",
		"some-json": `{    "glossary": {        "title": "example glossary",		"GlossDiv": {            "title": "S",			"GlossList": {                "GlossEntry": {                    "ID": "SGML",					"SortAs": "SGML",					"GlossTerm": "Standard Generalized Markup Language",					"Acronym": "SGML",					"Abbrev": "ISO 8879:1986",					"GlossDef": {                        "para": "A meta-markup language, used to create markup languages such as DocBook.",						"GlossSeeAlso": ["GML", "XML"]                    },					"GlossSee": "markup"                }            }        }    }}`,
	}
	envVars := map[string]string{}

	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: "env-test-annotations",
		Network:      "func_functions",
		FProcess:     "env",
		Annotations:  wantedAnnotations,
		EnvVars:      envVars,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	_ = invoke(t, functionRequest.FunctionName, emptyQueryString, http.StatusOK)
	function := get(t, functionRequest.FunctionName)
	if err := strMapEqual("annotations", *function.Annotations, wantedAnnotations); err != nil {
		t.Fatal(err)
	}
}

func Test_ListNamespaces(t *testing.T) {
	expectedNamespaces := append(config.Namespaces, config.DefaultNamespace)
	actualNamespaces, err := config.Client.ListNamespaces(context.Background())

	if err != nil {
		t.Fatalf("Unable to List OpenFaaS Namespaces: %q", err)
	}

	expectedLen := len(expectedNamespaces)
	actualLen := len(actualNamespaces)
	if expectedLen != actualLen {
		t.Fatalf("want %d namespace(s),  got %d namespace(s)", expectedLen, actualLen)
	}

	sort.Strings(expectedNamespaces)
	sort.Strings(actualNamespaces)

	for i, ns := range expectedNamespaces {
		if ns != actualNamespaces[i] {
			t.Fatalf("want namespace: %q , got %q", expectedNamespaces, actualNamespaces)
		}
	}
}

func strMapEqual(mapName string, got map[string]string, wanted map[string]string) error {
	// Can't assert length is equal as some providers i.e. faas-swarm add their own labels during
	// deployment like 'com.openfaas.function' and 'function'

	for k, v := range wanted {
		if _, ok := got[k]; !ok {
			return fmt.Errorf("got missing key, wanted %s %s", k, mapName)
		}

		if got[k] != v {
			return fmt.Errorf("got %s, wanted %s %s", got[k], v, mapName)
		}
	}

	return nil
}
