package tests

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	types "github.com/openfaas/faas-provider/types"
)

var emptyQueryString = ""

type FunctionTestCase struct {
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	imagePath := config.RegistryPrefix + "/" + "functions/alpine:latest"

	cases := []FunctionTestCase{
		{
			name: "Deploy without any extra metadata",
			function: types.FunctionDeployment{
				Image:       imagePath,
				Service:     "stronghash",
				EnvProcess:  "sha512sum",
				Annotations: &map[string]string{},
				Labels:      &map[string]string{},
				Namespace:   config.DefaultNamespace,
			},
		},
		{
			name: "Deploy with labels",
			function: types.FunctionDeployment{
				Image:       imagePath,
				Service:     "env-test-labels",
				EnvProcess:  "env",
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
				Image:      imagePath,
				Service:    "env-test-annotations",
				EnvProcess: "env",
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
	cases = copyNamespacesTest(cases)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			functionRequest := createDeploymentSpec(c)

			deployStatus := deploy(t, functionRequest)
			if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
				t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
			}

			function := get(t, functionRequest.FunctionName, functionRequest.Namespace)
			list(t, http.StatusOK, functionRequest.Namespace)
			if err := strMapEqual("annotations", *function.Annotations, *c.function.Annotations); err != nil {
				t.Fatal(err)
			}
			if err := strMapEqual("labels", *function.Labels, *c.function.Labels); err != nil {
				t.Fatal(err)
			}
		})
	}

	t.Run("test listing functions", func(t *testing.T) {

		listCases := map[string]map[string]types.FunctionDeployment{}
		for _, tc := range cases {
			fncs, ok := listCases[tc.function.Namespace]
			if !ok {
				fncs = map[string]types.FunctionDeployment{}
			}

			fncs[tc.function.Service] = tc.function
			listCases[tc.function.Namespace] = fncs
		}

		for namespace, expected := range listCases {
			actual, err := config.Client.ListFunctions(ctx, namespace)
			if err != nil {
				t.Fatalf("unable to List function in namspace: %s", err)
			}

			for _, actualF := range actual {
				expectedF, ok := expected[actualF.Name]
				if !ok {
					t.Fatalf("unexpected deployment %s found", actualF.Name)
				}

				err = compareDeployAndStatus(expectedF, actualF)
				if err != nil {
					t.Fatal(err)
				}
				delete(expected, actualF.Name)
			}

			if len(expected) != 0 {
				remaining := []string{}
				for name := range expected {
					remaining = append(remaining, name)
				}
				t.Fatalf("not all functions found in response: %s", strings.Join(remaining, ", "))
			}
		}
	})
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

func compareDeployAndStatus(deploy types.FunctionDeployment, status types.FunctionStatus) error {
	if deploy.Service != status.Name {
		return fmt.Errorf("got %v, expected name %s", status.Name, deploy.Service)
	}
	if deploy.Image != status.Image {
		return fmt.Errorf("got %v, expected image %s", status.Image, deploy.Image)
	}
	if deploy.Namespace != status.Namespace {
		return fmt.Errorf("got %v, expected Namespace %s", status.Namespace, deploy.Namespace)
	}
	if deploy.EnvProcess != status.EnvProcess {
		return fmt.Errorf("got %v, expected EnvProcess %s", status.EnvProcess, deploy.EnvProcess)
	}

	if deploy.ReadOnlyRootFilesystem != status.ReadOnlyRootFilesystem {
		return fmt.Errorf("got %v, expected ReadOnlyRootFilesystem %v", status.ReadOnlyRootFilesystem, deploy.ReadOnlyRootFilesystem)
	}

	if !reflect.DeepEqual(deploy.EnvVars, status.EnvVars) {
		return fmt.Errorf("got %v, expected EnvVars %v", status.EnvVars, deploy.EnvVars)
	}

	err := strSliceEqual(deploy.Constraints, status.Constraints)
	if err != nil {
		return fmt.Errorf("incorrect Constraints: %s", err)
	}

	err = strSliceEqual(deploy.Secrets, status.Secrets)
	if err != nil {
		return fmt.Errorf("incorrect Secrets: %s", err)
	}

	if !reflect.DeepEqual(deploy.Limits, status.Limits) {
		return fmt.Errorf("got %v, expected Limits %v", status.Limits, deploy.Limits)
	}

	if !reflect.DeepEqual(deploy.Requests, status.Requests) {
		return fmt.Errorf("got %v, expected Requests %v", status.Requests, deploy.Requests)
	}

	if config.ProviderName != faasdProviderName {
		// we expect all systems to add the `faas_function` label?
		expectedLabels := copyStrMap(deploy.Labels)
		expectedLabels["faas_function"] = deploy.Service
		if status.Labels == nil {
			return fmt.Errorf("lables should not be nil")
		}

		err = strMapEqual("Lables", *status.Labels, expectedLabels)
		if err != nil {
			return err
		}

		// some systems add additional annotations, we remove those
		if deploy.Annotations != nil && len(*deploy.Annotations) > 0 {
			if status.Annotations == nil {
				return fmt.Errorf("got nil Annotations, expected %d", len(*deploy.Annotations))
			}
			return strMapEqual("Annotations", *status.Annotations, *deploy.Annotations)
		}
	}

	return nil
}

func strMapEqual(mapName string, got map[string]string, wanted map[string]string) error {
	// Can't assert length is equal as some providers add their own labels during
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

func strSliceEqual(got, wanted []string) error {
	if len(got) != len(wanted) {
		return fmt.Errorf("incorrect number of entries")
	}
	for idx, value := range got {
		if wanted[idx] != value {
			return fmt.Errorf("got %s in position %d, expected %s,", value, idx, wanted[idx])
		}
	}
	return nil
}

func copyStrMap(src *map[string]string) map[string]string {
	dst := map[string]string{}
	if src == nil {
		return dst
	}

	for name, value := range *src {
		dst[name] = value
	}

	return dst
}
