package tests

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"testing"

	sdk "github.com/openfaas/faas-cli/proxy"
	types "github.com/openfaas/faas-provider/types"
)

var emptyQueryString = ""

type FunctionMetaSchema struct {
	name     string
	function types.FunctionDeployment
}

const someAnnotationJson = `{
	"glossary":{
	   "title":"example glossary",
	   "GlossDiv":{
		  "title":"S",
		  "GlossList":{
			 "GlossEntry":{
				"ID":"SGML",
				"SortAs":"SGML",
				"GlossTerm":"Standard Generalized Markup Language",
				"Acronym":"SGML",
				"Abbrev":"ISO 8879:1986",
				"GlossDef":{
				   "para":"A meta-markup language, used to create markup languages such as DocBook.",
				   "GlossSeeAlso":[
					  "GML",
					  "XML"
				   ]
				},
				"GlossSee":"markup"
			 }
		  }
	   }
	}
 }`

func Test_Deploy_MetaData(t *testing.T) {
	cases := []FunctionMetaSchema{
		{
			name: "Deploy without any extra metadata",
			function: types.FunctionDeployment{
				Image:       "functions/alpine:latest",
				Service:     "stronghash",
				EnvProcess:  "sha512sum",
				EnvVars:     map[string]string{},
				Annotations: &map[string]string{},
				Labels:      &map[string]string{},
				Namespace:   config.DefaultNamespace,
			},
		},
		{
			name: "Deploy with labels",
			function: types.FunctionDeployment{
				Image:       "functions/alpine:latest",
				Service:     "env-test-labels",
				EnvProcess:  "env",
				EnvVars:     map[string]string{},
				Annotations: &map[string]string{},
				Labels: &map[string]string{
					"upstream_uri": "example.com",
					"canary_build": "true",
				},
				Namespace: config.DefaultNamespace,
			},
		},
		{
			name: "Deploy with annotations",
			function: types.FunctionDeployment{
				Image:      "functions/alpine:latest",
				Service:    "env-test-annotations",
				EnvProcess: "env",
				EnvVars:    map[string]string{},
				Annotations: &map[string]string{
					"important-date": "Fri Aug 10 08:21:00 BST 2018",
					"some-json":      someAnnotationJson,
				},
				Labels:    &map[string]string{},
				Namespace: config.DefaultNamespace,
			},
		},
	}

	// Add Test case, if CERTIFIER_NAMESPACES defined
	if len(config.Namespaces) > 0 {
		cnCases := make([]FunctionMetaSchema, len(cases))
		copy(cnCases, cases)
		for index := 0; index < len(cnCases); index++ {
			cnCases[index].name = fmt.Sprintf("%s to %s", cnCases[index].name, config.Namespaces[0])
			cnCases[index].function.Namespace = config.Namespaces[0]
		}

		cases = append(cases, cnCases...)
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			functionRequest := &sdk.DeployFunctionSpec{
				Image:        c.function.Image,
				FunctionName: c.function.Service,
				FProcess:     c.function.EnvProcess,
				Annotations:  *c.function.Annotations,
				EnvVars:      c.function.EnvVars,
				Labels:       *c.function.Labels,
				Namespace:    c.function.Namespace,
			}

			deployStatus := deploy(t, functionRequest)
			if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
				t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
			}

			function := get(t, functionRequest.FunctionName)
			list(t, http.StatusOK)
			if err := strMapEqual("annotations", *function.Annotations, *c.function.Annotations); err != nil {
				t.Fatal(err)
			}
			if err := strMapEqual("labels", *function.Labels, *c.function.Labels); err != nil {
				t.Fatal(err)
			}
		})
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
