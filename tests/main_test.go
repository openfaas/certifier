package tests

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdkConfig "github.com/openfaas/faas-cli/config"

	sdk "github.com/openfaas/faas-cli/proxy"
)

var (
	config            = Config{}
	token             = flag.String("token", "", "authentication Bearer token override, enables auth automatically")
	faasdProviderName = "faasd"
	// faasNetesProviderName = "faas-netes"
)

func init() {

	flag.StringVar(&config.Gateway, "gateway", "", "set the gateway URL, if empty use the gateway_url env variable")

	flag.BoolVar(
		&config.AuthEnabled,
		"enableAuth",
		false,
		fmt.Sprintf("enable/disable authentication. The auth will be parsed from the default config in %s", filepath.Join(sdkConfig.DefaultDir, sdkConfig.DefaultFile)),
	)
	flag.BoolVar(&config.SecretUpdate, "secretUpdate", true, "enable/disable secret update tests")
	flag.BoolVar(&config.EnableScaling, "enableScaling", true, "enable/disable scale  tests")
	flag.StringVar(&config.RegistryPrefix, "registryPrefix", "docker.io", "provide custom registry path")

	FromEnv(&config)
}

func TestMain(m *testing.M) {
	// flag parsing here
	var err error
	flag.Parse()

	// get the gateway from the env
	if config.Gateway == "" {
		config.Gateway = os.Getenv("gateway_url")
	}

	// or use the default if it is still empty
	if config.Gateway == "" {
		config.Gateway = "http://127.0.0.1:8080/"
	}

	uri, err := url.Parse(config.Gateway)
	if err != nil {
		log.Fatalf("invalid gateway url %s", err)
	}

	config.Gateway = uri.String()

	// make sure to trim any trailing slash because this is how the gateway is modified when
	// saved to the config. if we don't do this, we wont find the saved auth.
	config.Gateway = strings.TrimRight(config.Gateway, "/")

	config.Auth = &Unauthenticated{}
	if config.AuthEnabled || *token != "" {
		// TODO : NewCLIAuth should return the error from LookupAuthConfig!
		config.Auth, err = sdk.NewCLIAuth(*token, config.Gateway)
		if err != nil {
			log.Fatalf("can not build cli auth: %s", err)
		}
	}

	timeout := 30 * time.Second
	config.Client, err = sdk.NewClient(config.Auth, config.Gateway, nil, &timeout)
	if err != nil {
		log.Fatalf("can not client: %s", err)
	}

	config.ProviderName, err = getProvider(config.Client)
	if err != nil {
		log.Fatalf("Can not get system info: %s", err)
	}

	if config.ProviderName == faasdProviderName {
		config.EnableScaling = false
		config.SecretUpdate = false
	}

	config.SupportCPULimits = config.ProviderName != faasdProviderName

	prettyConfig, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		log.Fatalf("Config Pretty Print Failed with %s", err)
	}
	log.Println(string(prettyConfig))

	os.Exit(m.Run())
}

// Config contains the configuration values for the certifier tests
// This includes the gateway and auth parameters as well as the feature
// flags to control skipping specific tests.
type Config struct {
	// Gateway is the URL for the gateway that will be tested
	Gateway string
	// Auth contains the parsed proxy client auth
	Auth sdk.ClientAuth
	// Client is a preconfigured gateway client, including auth
	Client *sdk.Client

	// AuthEnabled
	AuthEnabled bool

	// SecretUpdate enables/disables the secret update test
	SecretUpdate bool

	// Namespaces to verfiy OpenFaaS provider
	Namespaces []string

	// DefaultNamespace for OpenFaas provider
	DefaultNamespace string

	// Provider Name of Openfaas
	ProviderName string
	//EnableScale will enable scaling test cases
	EnableScaling bool

	// registry prefix for private registry
	RegistryPrefix string

	SupportCPULimits bool
}

func FromEnv(config *Config) {
	// read CERTIFIER_NAMESPACES variable, parse as csv string
	namespaces, present := os.LookupEnv("CERTIFIER_NAMESPACES")
	if present {
		config.Namespaces = strings.Split(namespaces, ",")
		for index := range config.Namespaces {
			config.Namespaces[index] = strings.TrimSpace(config.Namespaces[index])
		}

		// filter empty values from config.Namespaces in place
		n := 0
		for _, x := range config.Namespaces {
			if x != "" {
				config.Namespaces[n] = x
				n++
			}
		}
		config.Namespaces = config.Namespaces[:n]
	}

	// read CERTIFIER_DEFAULT_NAMESPACE variable, if not apply openfaas-fn
	defaultNamespace, present := os.LookupEnv("CERTIFIER_DEFAULT_NAMESPACE")

	if present && strings.TrimSpace(defaultNamespace) != "" {
		config.DefaultNamespace = defaultNamespace
	} else {
		config.DefaultNamespace = "openfaas-fn"
	}
}

func getProvider(client *sdk.Client) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := client.GetSystemInfo(ctx)
	if err != nil {
		return "", err
	}

	return info.Provider.Name, nil
}
