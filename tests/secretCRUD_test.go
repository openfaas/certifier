package tests

import (
	"encoding/json"
	"flag"
	"net/http"
	"testing"

	"github.com/openfaas/faas-provider/types"
	"github.com/openfaas/faas/gateway/requests"
)

var secretsPath = "system/secrets"
var swarm = flag.Bool("swarm", false, "run swarm-compatible tests only")

type secret types.Secret

func Test_SecretCRUD(t *testing.T) {
	setValue := "this-is-the-secret-value"
	setName := "secret-name"
	functionName := "test-secret-crud"

	createStatus := createSecret(t, setName, setValue)
	if createStatus != http.StatusCreated && createStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", createStatus, http.StatusOK, http.StatusAccepted)
	}
	t.Logf("Got correct response for creating secret: %d", createStatus)

	// Set up and deploy function that reads the value of the created secret.
	functionRequest := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    functionName,
		Network:    "func_functions",
		EnvProcess: "cat /var/openfaas/secrets/" + setName,
		Secrets:    []string{setName},
	}
	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Errorf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}
	t.Logf("Got correct response for deploying function: %d", deployStatus)

	// Verify that the secret value was set as intended.
	value := string(invoke(t, functionRequest.Service, "", http.StatusOK))
	if value != setValue {
		t.Errorf("got %s, wanted %s", value, setValue)
	}

	// Verify that the secret can be listed.
	names := listSecrets(t)
	if !listContains(names, setName) {
		t.Errorf("got %v, wanted %s in slice", names, setName)
	}

	// Docker Swarm secrets are immutable, so skip the update tests for swarm.
	if !*swarm {
		newValue := "this-is-the-edited-secret-value"
		updateStatus := updateSecret(t, setName, newValue)
		if updateStatus != http.StatusOK && updateStatus != http.StatusAccepted {
			t.Errorf("got %d, wanted %d or %d", updateStatus, http.StatusOK, http.StatusAccepted)
		}
		t.Logf("Got correct response for updating secret: %d", updateStatus)

		// Verify that the secret value was edited.
		value = string(invoke(t, functionRequest.Service, "", http.StatusOK))
		if value != setValue {
			t.Errorf("got %s, wanted %s", value, newValue)
		}
	}

	// Function needs to be deleted to free up the secret so it can also be deleted.
	delFunctionRequest := requests.DeleteFunctionRequest{
		FunctionName: functionName,
	}

	deleteResp, res := request(t, gatewayUrl(t, "system/functions", ""), http.MethodDelete, makeReader(delFunctionRequest))
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted {
		t.Errorf("got %d, wanted %d or %d: %s", res.StatusCode, http.StatusOK, http.StatusAccepted, deleteResp)
	}
	t.Logf("Got correct response for deleting function")

	deleteStatus := deleteSecret(t, setName)
	if deleteStatus != http.StatusOK && deleteStatus != http.StatusAccepted {
		t.Errorf("got %d, wanted %d or %d", deleteStatus, http.StatusOK, http.StatusAccepted)
	}
	t.Logf("Got correct response for deleting secret: %d", deleteStatus)

	// Verify that the secret was deleted.
	names = listSecrets(t)
	if listContains(names, setName) {
		t.Errorf("got %v, wanted %s deleted", names, setName)
	}
}

func createSecret(t *testing.T, name, value string) int {
	t.Helper()

	req := secret{
		Name:  name,
		Value: value,
	}
	rdr := makeReader(req)

	secretsURL := gatewayUrl(t, secretsPath, "")
	_, res := request(t, secretsURL, http.MethodPost, rdr)
	return res.StatusCode
}

func updateSecret(t *testing.T, name, edit string) int {
	req := secret{
		Name:  name,
		Value: edit,
	}
	rdr := makeReader(req)

	secretsURL := gatewayUrl(t, secretsPath, "")
	_, res := request(t, secretsURL, http.MethodPut, rdr)
	return res.StatusCode
}

func deleteSecret(t *testing.T, name string) int {
	req := secret{Name: name}
	rdr := makeReader(req)

	secretsURL := gatewayUrl(t, secretsPath, "")
	_, res := request(t, secretsURL, http.MethodDelete, rdr)
	return res.StatusCode
}

func listContains(list []secret, s string) bool {
	for i := range list {
		if list[i].Name == s {
			return true
		}
	}
	return false
}

func listSecrets(t *testing.T) []secret {
	secretsList := []secret{}

	secretsURL := gatewayUrl(t, secretsPath, "")
	secrets, res := request(t, secretsURL, http.MethodGet, nil)

	err := json.Unmarshal(secrets, &secretsList)
	if err != nil {
		t.Fatalf("unable to parse respose: %s", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200 when listing secrets")
	}

	t.Logf("Got correct response for listing secret: %d", res.StatusCode)
	return secretsList
}
