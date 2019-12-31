package tests

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/openfaas/faas-provider/types"
	"github.com/openfaas/faas/gateway/requests"
)

var secretsURL string = os.Getenv("gateway_url") + "system/secrets"
var swarm = flag.Bool("swarm", false, "run swarm-compatible tests only")

type secret types.Secret

func Test_SecretCRUD(t *testing.T) {
	setValue := "this-is-the-secret-value"
	setName := "secret-name"
	functionName := "test-secret-crud"

	createStatus, err := createSecret(setName, setValue)
	if err != nil {
		t.Error(err)
	}
	if createStatus != http.StatusCreated && createStatus != http.StatusAccepted {
		t.Errorf("got %d, wanted %d or %d", createStatus, http.StatusOK, http.StatusAccepted)
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
	deployStatus, err := deploy(t, functionRequest)
	if err != nil {
		t.Error(err)
	}
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
	names, listStatus, err := listSecrets()
	if err != nil {
		t.Error(err)
	}
	if !listContains(names, setName) {
		t.Errorf("got %v, wanted %s in slice", names, setName)
	}
	t.Logf("Got correct response for listing secrets: %d", listStatus)

	// Docker Swarm secrets are immutable, so skip the update tests for swarm.
	if !*swarm {
		newValue := "this-is-the-edited-secret-value"
		updateStatus, err := updateSecret(setName, newValue)
		if err != nil {
			t.Error(err)
		}
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
	delFunctionStatus, err := delete(t, delFunctionRequest)
	if err != nil {
		t.Error(err)
	}
	if delFunctionStatus != http.StatusOK && delFunctionStatus != http.StatusAccepted {
		t.Errorf("got %d, wanted %d or %d", delFunctionStatus, http.StatusOK, http.StatusAccepted)
	}
	t.Logf("Got correct response for deleting function: %d", delFunctionStatus)

	deleteStatus, err := deleteSecret(setName)
	if err != nil {
		t.Error(err)
	}
	if deleteStatus != http.StatusOK && deleteStatus != http.StatusAccepted {
		t.Errorf("got %d, wanted %d or %d", deleteStatus, http.StatusOK, http.StatusAccepted)
	}
	t.Logf("Got correct response for deleting secret: %d", deleteStatus)

	// Verify that the secret was deleted.
	names, listStatus, err = listSecrets()
	if err != nil {
		t.Error(err)
	}
	if listContains(names, setName) {
		t.Errorf("got %v, wanted %s deleted", names, setName)
	}
	t.Logf("Got correct response for listing secret: %d", listStatus)
}

func createSecret(name, value string) (int, error) {
	req := secret{
		Name:  name,
		Value: value,
	}
	rdr := makeReader(req)

	_, res, err := httpReq(secretsURL, http.MethodPost, rdr)
	return res.StatusCode, err
}

func updateSecret(name, edit string) (int, error) {
	req := secret{
		Name:  name,
		Value: edit,
	}
	rdr := makeReader(req)

	_, res, err := httpReq(secretsURL, http.MethodPut, rdr)
	return res.StatusCode, err
}

func deleteSecret(name string) (int, error) {
	req := secret{Name: name}
	rdr := makeReader(req)

	_, res, err := httpReq(secretsURL, http.MethodDelete, rdr)

	if err != nil {
		return 0, err
	}
	return res.StatusCode, nil
}

func listContains(list []secret, s string) bool {
	for i := range list {
		if list[i].Name == s {
			return true
		}
	}
	return false
}

func listSecrets() ([]secret, int, error) {
	secretsList := []secret{}

	secrets, res, err := httpReq(secretsURL, http.MethodGet, nil)
	if err != nil {
		return secretsList, res.StatusCode, err
	}

	json.Unmarshal(secrets, &secretsList)
	return secretsList, res.StatusCode, nil
}

func delete(t *testing.T, delFunctionRequest requests.DeleteFunctionRequest) (int, error) {
	body, res, err := httpReq(os.Getenv("gateway_url")+"system/functions", http.MethodDelete, makeReader(delFunctionRequest))

	if err != nil {
		return http.StatusBadGateway, err
	}

	if res.StatusCode >= 400 {
		t.Logf("Delete response: %s", string(body))
		return res.StatusCode, fmt.Errorf("unable to delete function")
	}
	return res.StatusCode, nil
}
