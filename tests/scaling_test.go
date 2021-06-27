package tests

import (
	"bytes"
	"fmt"
	"net/http"
	"path"
	"testing"
	"time"

	sdk "github.com/openfaas/faas-cli/proxy"
	"github.com/rakyll/hey/requester"
)

type scalingTestCase struct {
	name           string
	spec           sdk.DeployFunctionSpec
	minReplicas    int
	maxReplicas    int
	targetReplicas int
	withLoad       bool
	scaleFromZero  bool
}

func (t *scalingTestCase) SetNamespace(namespace string) {
	t.name = fmt.Sprintf("%s in %s", t.name, namespace)
	t.spec.Namespace = namespace
}

func Test_Scaling(t *testing.T) {
	cases := []scalingTestCase{
		{
			name: "deploy with non-default minimum replicas",
			spec: sdk.DeployFunctionSpec{
				Image:        "functions/alpine:latest",
				FunctionName: "test-min-scale",
				Network:      "func_functions",
				FProcess:     "sha512sum",
				Labels: map[string]string{
					"com.openfaas.scale.min": fmt.Sprintf("%d", uint64(2)),
				},
				Namespace: config.DefaultNamespace,
			},
			minReplicas: 2,
		},
		{
			name: "scale up from zero replicas after invoke",
			spec: sdk.DeployFunctionSpec{
				Image:        "functions/alpine:latest",
				FunctionName: "test-scale-from-zero",
				Network:      "func_functions",
				FProcess:     "sha512sum",
				Namespace:    config.DefaultNamespace,
			},
			scaleFromZero:  true,
			targetReplicas: 1,
			minReplicas:    1,
		},
		{
			name: "scale up and down via load monitoring",
			spec: sdk.DeployFunctionSpec{
				Image:        "functions/alpine:latest",
				FunctionName: "test-throughput-scaling",
				Network:      "func_functions",
				FProcess:     "sha512sum",
				Labels: map[string]string{
					"com.openfaas.scale.min": fmt.Sprintf("%d", 1),
					"com.openfaas.scale.max": fmt.Sprintf("%d", 2),
				},
				Namespace: config.DefaultNamespace,
			},
			minReplicas:    1,
			maxReplicas:    2,
			targetReplicas: 1,
			withLoad:       true,
		},
		{
			name: "scale to zero",
			spec: sdk.DeployFunctionSpec{
				Image:        "functions/alpine:latest",
				FunctionName: "test-scaling-to-zero",
				Network:      "func_functions",
				FProcess:     "sha512sum",
				Labels: map[string]string{
					"com.openfaas.scale.max":  fmt.Sprintf("%d", 2),
					"com.openfaas.scale.zero": "true",
				},
				Namespace: config.DefaultNamespace,
			},
			minReplicas:    1,
			maxReplicas:    2,
			targetReplicas: 0,
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

			if !config.IdlerEnabled && tc.spec.Labels["com.openfaas.scale.zero"] == "true" {
				t.Skip("set 'idler_enabled' to test scale to zero")
				return
			}

			deployStatus := deploy(t, &tc.spec)
			if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
				t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
			}

			defer deleteFunction(t, &tc.spec)

			time.Sleep(5 * time.Second)
			fnc := get(t, tc.spec.FunctionName, tc.spec.Namespace)
			if fnc.Replicas != uint64(tc.minReplicas) {
				t.Fatalf("got %d replicas, wanted %d", fnc.Replicas, tc.minReplicas)
			}

			if tc.scaleFromZero {
				scaleFunction(t, tc.spec.FunctionName, tc.spec.Namespace, 0)

				fnc := get(t, tc.spec.FunctionName, tc.spec.Namespace)
				if fnc.Replicas != 0 {
					t.Fatalf("got %d replicas, wanted %d", fnc.Replicas, 0)
				}

				// this will fail or pass the test
				_ = invoke(t, &tc.spec, "", "", http.StatusOK)
			}

			if tc.withLoad {
				functionURL := resourceURL(t, path.Join("function", tc.spec.FunctionName+"."+tc.spec.Namespace), "")
				req, err := http.NewRequest(http.MethodPost, functionURL, nil)
				if err != nil {
					t.Fatalf("error with request %s ", err)
				}

				var loadOutput bytes.Buffer
				attempts := 1000
				functionLoad := requester.Work{
					Request:           req,
					N:                 attempts,
					Timeout:           10,
					C:                 2,
					QPS:               5.0,
					DisableKeepAlives: true,
					Writer:            &loadOutput,
				}

				functionLoad.Run()

				fnc := get(t, tc.spec.FunctionName, tc.spec.Namespace)
				if fnc.Replicas != uint64(tc.maxReplicas) {
					t.Logf("function load output %s", loadOutput.String())
					t.Fatalf("never reached max scale %d, only %d replicas after %d attempts", tc.maxReplicas, fnc.Replicas, attempts)
				}

				// no need to test cooldown if min=max, because it is effectively tested above
				if tc.maxReplicas > tc.minReplicas {
					time.Sleep(time.Minute)
					fnc = get(t, tc.spec.FunctionName, tc.spec.Namespace)
					if fnc.Replicas != uint64(tc.targetReplicas) {
						t.Fatalf("got %d replicas, wanted %d", fnc.Replicas, tc.targetReplicas)
					}
				}
			}
		})
	}
}
