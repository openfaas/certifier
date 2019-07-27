package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/openfaas/faas/gateway/requests"
)

var emptyQueryString = ""

func Test_Access_Secret(t *testing.T) {
	secret := os.Getenv("SECRET")
	secrets := []string{"secret-api-test-key"}
	functionRequest := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    "test-secret",
		Network:    "func_functions",
		EnvProcess: "cat /var/openfaas/secrets/secret-api-test-key",
		Secrets:    secrets,
	}

	deployStatus, deployErr := deploy(t, functionRequest)
	if deployErr != nil {
		t.Errorf(deployErr.Error())
		t.Fail()
		return
	}

	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Errorf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	list(t, http.StatusOK)

	t.Run("Empty QueryString", func(t *testing.T) {

		bytesOut := invoke(t, functionRequest.Service, emptyQueryString, http.StatusOK)

		out := strings.TrimSuffix(string(bytesOut), "\n")
		if out != secret {
			t.Errorf("want: %q, got: %q", secret, out)
		}
	})
}

func Test_Deploy_Stronghash(t *testing.T) {
	envVars := map[string]string{}
	functionRequest := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    "stronghash",
		Network:    "func_functions",
		EnvProcess: "sha512sum",
		EnvVars:    envVars,
	}

	deployStatus, deployErr := deploy(t, functionRequest)
	if deployErr != nil {
		t.Errorf(deployErr.Error())
		t.Fail()
		return
	}

	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Logf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
		t.Fail()
	}

	list(t, http.StatusOK)
}

func Test_Deploy_PassingCustomEnvVars_AndQueryString(t *testing.T) {
	envVars := map[string]string{}
	envVars["custom_env"] = "custom_env_value"

	functionRequest := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    "env-test",
		Network:    "func_functions",
		EnvProcess: "env",
		EnvVars:    envVars,
	}

	deployStatus, deployErr := deploy(t, functionRequest)
	if deployErr != nil {
		t.Errorf(deployErr.Error())
		t.Fail()
		return
	}

	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Logf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
		t.Fail()
	}

	list(t, http.StatusOK)

	t.Run("Empty QueryString", func(t *testing.T) {
		bytesOut := invoke(t, functionRequest.Service, emptyQueryString, http.StatusOK)

		out := string(bytesOut)
		if strings.Contains(out, "custom_env") == false {
			t.Logf("want: %s, got: %s", "custom_env", out)
			t.Fail()
		}
	})

	t.Run("Populated QueryString", func(t *testing.T) {
		bytesOut := invoke(t, functionRequest.Service, "testing=1", http.StatusOK)

		out := string(bytesOut)
		if strings.Contains(out, "Http_Query=testing=1") == false {
			t.Logf("want: %s, got: %s", "Http_Query=testing=1", out)
			t.Fail()
		}
	})
}

func Test_Auto_Scaling(t *testing.T) {
	labels := map[string]string{
		"com.openfaas.scale.min": "1",
		"com.openfaas.scale.max": "2",
	}

	functionRequest := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    "test-auto-scaling",
		Network:    "func_functions",
		EnvProcess: "env",
		Labels:     &labels,
	}

	deployStatus, deployErr := deploy(t, functionRequest)
	if deployErr != nil {
		t.Errorf(deployErr.Error())
		t.Fail()
		return
	}

	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Logf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
		t.Fail()
	}

	finishHTTP := make(chan bool)

	go func() {
		end := false
		for !end {
			select {
			case <-finishHTTP:
				end = true
			default:
				invoke(t, functionRequest.Service, emptyQueryString, http.StatusOK)
			}
		}
	}()

	// calling function for 60 seconds to increase # of replicas
	endSignal := time.After(60 * time.Second)
	end := false

	for !end {
		select {
		case <-endSignal:
			t.Logf("expected 2 replicas not 1 after calling function for 60 seconds")
			t.Fail()
			finishHTTP <- true
			end = true
		default:
			function := get(t, functionRequest.Service)

			if function.Replicas == 2 {
				finishHTTP <- true
				end = true
			} else {
				time.Sleep(time.Millisecond * 1000)
			}
		}
	}

	// waiting for replica count to go down
	endSignal = time.After(60 * time.Second)
	end = false

	for !end {
		select {
		case <-endSignal:
			t.Logf("expected 1 replica not 2 after not calling function for 60 seconds")
			t.Fail()
			end = true
		default:
			function := get(t, functionRequest.Service)

			if function.Replicas == 1 {
				end = true
			} else {
				time.Sleep(time.Millisecond * 1000)
			}
		}
	}
}

func Test_Scale_From_Zero(t *testing.T) {

	functionRequest := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    "test-scale-zero",
		Network:    "func_functions",
		EnvProcess: "env",
	}

	deployStatus, deployErr := deploy(t, functionRequest)
	if deployErr != nil {
		t.Errorf(deployErr.Error())
		t.Fail()
		return
	}

	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Logf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
		t.Fail()
	}

	scale(t, "test-scale-zero", 0)
	function := get(t, functionRequest.Service)

	if function.Replicas != 0 {
		t.Logf("want: %d replicas, got: %d replicas", 0, function.Replicas)
		t.Fail()
	}

	invoke(t, functionRequest.Service, emptyQueryString, http.StatusOK)
}

func Test_Deploy_WithMinScale(t *testing.T) {
	labels := map[string]string{
		"com.openfaas.scale.min": "2",
	}

	functionRequest := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    "test-min-replica",
		Network:    "func_functions",
		EnvProcess: "env",
		Labels:     &labels,
	}

	deployStatus, deployErr := deploy(t, functionRequest)
	if deployErr != nil {
		t.Errorf(deployErr.Error())
		t.Fail()
		return
	}

	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Logf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
		t.Fail()
	}

	function := get(t, functionRequest.Service)

	if function.Replicas != 2 {
		t.Logf("want: %d replicas, got: %d replicas", 2, function.Replicas)
		t.Fail()
	}
}

func Test_Deploy_WithLabels(t *testing.T) {
	wantedLabels := map[string]string{
		"upstream_uri": "example.com",
		"canary_build": "true",
	}
	envVars := map[string]string{}

	functionRequest := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    "env-test-labels",
		Network:    "func_functions",
		EnvProcess: "env",
		Labels:     &wantedLabels,
		EnvVars:    envVars,
	}

	deployStatus, deployErr := deploy(t, functionRequest)
	if deployErr != nil {
		t.Errorf(deployErr.Error())
		t.Fail()
		return
	}

	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Logf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
		t.Fail()
	}

	invoke(t, functionRequest.Service, emptyQueryString, http.StatusOK)
	function := get(t, functionRequest.Service)

	if err := strMapEqual("labels", *function.Labels, wantedLabels); err != nil {
		t.Log(err)
		t.Fail()
	}
}

func Test_Deploy_WithAnnotations(t *testing.T) {
	wantedAnnotations := map[string]string{
		"important-date": "Fri Aug 10 08:21:00 BST 2018",
		"some-json": `{    "glossary": {        "title": "example glossary",		"GlossDiv": {            "title": "S",			"GlossList": {                "GlossEntry": {                    "ID": "SGML",					"SortAs": "SGML",					"GlossTerm": "Standard Generalized Markup Language",					"Acronym": "SGML",					"Abbrev": "ISO 8879:1986",					"GlossDef": {                        "para": "A meta-markup language, used to create markup languages such as DocBook.",						"GlossSeeAlso": ["GML", "XML"]                    },					"GlossSee": "markup"                }            }        }    }}`,
	}
	envVars := map[string]string{}

	functionRequest := requests.CreateFunctionRequest{
		Image:       "functions/alpine:latest",
		Service:     "env-test-annotations",
		Network:     "func_functions",
		EnvProcess:  "env",
		Annotations: &wantedAnnotations,
		EnvVars:     envVars,
	}

	deployStatus, deployErr := deploy(t, functionRequest)
	if deployErr != nil {
		t.Errorf(deployErr.Error())
		t.Fail()
		return
	}

	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Logf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
		t.Fail()
	}

	invoke(t, functionRequest.Service, emptyQueryString, http.StatusOK)
	function := get(t, functionRequest.Service)

	if err := strMapEqual("annotations", *function.Annotations, wantedAnnotations); err != nil {
		t.Log(err)
		t.Fail()
	}
}

func scale(t *testing.T, function string, replicas uint64) (int, error) {

	req := map[string]interface{}{"services": function, "replicas": replicas}

	body, res, err := httpReq(fmt.Sprintf("%ssystem/scale-function/%s",
		os.Getenv("gateway_url"), function), http.MethodPost, makeReader(req))

	if err != nil {
		return http.StatusBadGateway, err
	}

	if res.StatusCode >= 400 {
		t.Logf("Deploy response: %s", string(body))
		return res.StatusCode, fmt.Errorf("unable to deploy function")
	}

	return res.StatusCode, nil
}

func deploy(t *testing.T, createRequest requests.CreateFunctionRequest) (int, error) {
	body, res, err := httpReq(os.Getenv("gateway_url")+"system/functions", http.MethodPost, makeReader(createRequest))

	if err != nil {
		return http.StatusBadGateway, err
	}

	if res.StatusCode >= 400 {
		t.Logf("Deploy response: %s", string(body))
		return res.StatusCode, fmt.Errorf("unable to deploy function")
	}

	return res.StatusCode, nil
}

func list(t *testing.T, expectedStatusCode int) {

	bytesOut, res, err := httpReq(os.Getenv("gateway_url")+"system/functions", http.MethodGet, nil)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	if res.StatusCode != expectedStatusCode {
		t.Logf("got %d, wanted %d", res.StatusCode, expectedStatusCode)
		t.Fail()
	}

	functions := []requests.Function{}
	err = json.Unmarshal(bytesOut, &functions)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if len(functions) == 0 {
		t.Log("List functions got: 0, want: > 0")
		t.Fail()
	}
}

func get(t *testing.T, name string) requests.Function {

	bytesOut, res, err := httpReq(fmt.Sprintf("%ssystem/function/%s",
		os.Getenv("gateway_url"), name), http.MethodGet, nil)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	if res.StatusCode != 200 {
		t.Logf("got %d, wanted %d", res.StatusCode, 200)
		t.Fail()
	}

	function := requests.Function{}
	err = json.Unmarshal(bytesOut, &function)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	return function
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
