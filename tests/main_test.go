package tests

import (
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
	config           = Config{}
	defaultNamespace = ""
	swarm            = flag.Bool("swarm", false, "helper flag to run only swarm-compatible tests only")
	token            = flag.String("token", "", "authentication Bearer token override, enables auth automatically")
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
	flag.BoolVar(&config.ScaleToZero, "scaleToZero", true, "enable/disable scale from zero tests")

	FromEnv(&config)
}

func TestMain(m *testing.M) {
	// flag parsing here
	var err error
	flag.Parse()

	if config.Gateway == "" {
		uri, err := url.Parse(os.Getenv("gateway_url"))
		if err != nil {
			log.Fatalf("invalid gateway url %s", err)
		}

		config.Gateway = uri.String()
	}

	// make sure to trim any trailing slash because this is how the gateway is modified when
	// saved to the config. if we don't do this, we wont find the saved auth.
	config.Gateway = strings.TrimRight(config.Gateway, "/")

	if *swarm {
		config.SecretUpdate = false
		config.ScaleToZero = false
	}

	config.Auth = &Unauthenticated{}
	if config.AuthEnabled || *token != "" {
		// TODO : NewCLIAuth should return the error from LookupAuthConfig!
		config.Auth, err = sdk.NewCLIAuth(*token, config.Gateway)
		if err != nil {
			log.Fatalf("can not build cli auth: %s", err)
		}
	}

	timeout := 5 * time.Second
	config.Client, err = sdk.NewClient(config.Auth, config.Gateway, nil, &timeout)
	if err != nil {
		log.Fatalf("can not client: %s", err)
	}

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
	// ScaleToZero enables/disables the scale from zero test
	ScaleToZero bool

	// Namespaces to verfiy OpenFaaS provider
	Namespaces []string

	// DefaultNamespace for OpenFaas provider
	DefaultNamespace string
}

func FromEnv(config *Config) {
	// read CERTIFIER_NAMESPACES variable, parse as csv string
	namespaces, present := os.LookupEnv("CERTIFIER_NAMESPACES")
	if present {
		config.Namespaces = strings.Split(namespaces, ",")
		for index := range config.Namespaces {
			config.Namespaces[index] = strings.TrimSpace(config.Namespaces[index])
		}
	}

	// read CERTIFIER_DEFAULT_NAMESPACE variable, if not apply openfaas-fn
	defaultNamespace, present := os.LookupEnv("CERTIFIER_DEFAULT_NAMESPACE")

	if present {
		config.DefaultNamespace = defaultNamespace
	} else {
		config.DefaultNamespace = "openfaas-fn"
	}
}
