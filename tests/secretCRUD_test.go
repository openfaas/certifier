package tests

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	sdk "github.com/openfaas/faas-cli/proxy"
	"github.com/openfaas/faas-provider/types"
)

type secretTestCase struct {
	name         string
	secret       types.Secret
	secretUpdate types.Secret
}

func (t *secretTestCase) SetNamespace(namespace string) {
	t.name = fmt.Sprintf("%s in %s", t.name, namespace)
	t.secret.Namespace = namespace
	t.secretUpdate.Namespace = namespace
}

func Test_SecretCRUD(t *testing.T) {
	ctx := context.Background()

	cases := []secretTestCase{
		{
			name: "from string value",
			secret: types.Secret{
				Name:      "secret-string",
				Value:     "this-is-the-secret-string-value",
				Namespace: config.DefaultNamespace,
			},
			secretUpdate: types.Secret{
				Name:      "secret-string",
				Value:     "this-is-the-NEW-secret-string-value",
				Namespace: config.DefaultNamespace,
			},
		},
		{
			name: "from raw value",
			secret: types.Secret{
				Name:      "secret-bytes",
				RawValue:  []byte("this-is-the-RAW-secret-value"),
				Namespace: config.DefaultNamespace,
			},
			secretUpdate: types.Secret{
				Name:      "secret-bytes",
				RawValue:  []byte("this-is-the-NEW-RAW-secret-value"),
				Namespace: config.DefaultNamespace,
			},
		},
	}

	if len(config.Namespaces) > 0 {
		defaultCasesLen := len(cases)
		for index := 0; index < defaultCasesLen; index++ {
			namespacedCase := cases[index]
			namespacedCase.SetNamespace(config.Namespaces[0])
			cases = append(cases, namespacedCase)
		}
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			functionName := tc.secret.Name
			value := tc.secret.Value
			if tc.secret.Value == "" {
				value = string(tc.secret.RawValue)
			}

			// Verify that the secret are empty.
			secrets, err := config.Client.GetSecretList(ctx, tc.secret.Namespace)
			if err != nil {
				t.Fatal(err)
			}

			if listContains(secrets, tc.secret.Name) {
				t.Fatalf("namespace already has secret %s in %s: %v", tc.secret.Name, tc.secret.Namespace, secrets)
			}

			t.Logf("existing secrets in %s: %v", tc.secret.Namespace, secrets)

			// Set up and deploy function that reads the value of the created secret.
			functionRequest := &sdk.DeployFunctionSpec{
				Image:        "functions/alpine:latest",
				FunctionName: functionName,
				Network:      "func_functions",
				FProcess:     "cat /var/openfaas/secrets/" + tc.secret.Name,
				Secrets:      []string{tc.secret.Name},
				Namespace:    tc.secret.Namespace,
				Annotations:  map[string]string{},
			}

			t.Run("create", func(t *testing.T) {
				createStatus, _ := config.Client.CreateSecret(ctx, tc.secret)
				if createStatus != http.StatusCreated && createStatus != http.StatusAccepted {
					t.Fatalf("got %d, wanted %d or %d", createStatus, http.StatusOK, http.StatusAccepted)
				}

				deployStatus := deploy(t, functionRequest)
				if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
					t.Errorf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
				}

				// Verify that the secret value was set as intended.
				mountedValue := string(invoke(t, functionRequest, "", "", http.StatusOK))
				if mountedValue != value {
					t.Errorf("got %s, wanted %s", value, value)
				}
			})

			t.Run("list", func(t *testing.T) {
				// Verify that the secret can be listed.
				secrets, err := config.Client.GetSecretList(ctx, tc.secret.Namespace)
				if err != nil {
					t.Fatal(err)
				}

				if !listContains(secrets, tc.secret.Name) {
					t.Errorf("got %v, wanted %s in slice", secrets, tc.secret.Name)
				}
			})

			t.Run("update", func(t *testing.T) {
				if !config.SecretUpdate {
					// Docker Swarm secrets are immutable, so skip the update tests for swarm.
					t.Skip("secret update not enabled")
					return
				}

				value := tc.secretUpdate.Value
				if tc.secretUpdate.Value == "" {
					value = string(tc.secretUpdate.RawValue)
				}

				updateStatus, _ := config.Client.UpdateSecret(ctx, tc.secretUpdate)
				if updateStatus != http.StatusOK && updateStatus != http.StatusAccepted {
					t.Errorf("got %d, wanted %d or %d", updateStatus, http.StatusOK, http.StatusAccepted)
				}

				// let the cluster stabilize
				time.Sleep(time.Second)

				functionRequest.Update = true
				functionRequest.Annotations["secret-hash"] = "something to convince orchestrators that the deployment has changed"
				deployStatus := deploy(t, functionRequest)
				if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
					t.Errorf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
				}

				// let the cluster stabilize
				time.Sleep(5 * time.Second)

				// Verify that the secret value was updated and mounted
				mountedValue := string(invoke(t, functionRequest, "", "", http.StatusOK))
				if mountedValue != value {
					t.Errorf("got %s, wanted %s", mountedValue, value)
				}
			})

			t.Run("delete", func(t *testing.T) {
				// Function needs to be deleted to free up the secret so it can also be deleted.
				err := config.Client.DeleteFunction(ctx, functionRequest.FunctionName, functionRequest.Namespace)
				if err != nil {
					t.Fatal(err)
				}

				err = config.Client.RemoveSecret(ctx, tc.secret)
				if err != nil {
					t.Fatal(err)
				}

				// Verify that the secret was deleted.
				secrets, err := config.Client.GetSecretList(ctx, tc.secret.Namespace)
				if err != nil {
					t.Fatal(err)
				}

				if listContains(secrets, tc.secret.Name) {
					t.Errorf("got %v, wanted %s deleted", secrets, tc.secret.Name)
				}
			})
		})
	}
}

func listContains(list []types.Secret, s string) bool {
	for i := range list {
		if list[i].Name == s {
			return true
		}
	}
	return false
}
