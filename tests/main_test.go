package tests

import (
	"flag"
	"log"
	"net/url"
	"os"
	"testing"

	"github.com/openfaas/faas-cli/proxy"
)

var (
	config = Config{}
	swarm  = flag.Bool("swarm", false, "helper flag to run only swarm-compatible tests only")
)

func init() {

	flag.StringVar(&config.Gateway, "gateway", "", "set the gateway URL, if empty use the gateway_url env variable")

	flag.BoolVar(&config.SecretUpdate, "secretUpdate", true, "enable/disable secret update tests")
	flag.BoolVar(&config.ScaleToZero, "scaleToZero", true, "enable/disable scale from zero tests")
}

func TestMain(m *testing.M) {
	// flag parsing here
	flag.Parse()

	if config.Gateway == "" {
		uri, err := url.Parse(os.Getenv("gateway_url"))
		if err != nil {
			log.Fatalf("invalid gateway url %s", err)
		}

		config.Gateway = uri.String()
	}

	if *swarm {
		config.SecretUpdate = false
		config.ScaleToZero = false
	}

	// auth, err := cliConfig.LookupAuthConfig(config.Gateway)
	// if err != nil {
	// 	log.Fatalf("invalid gateway url %s", err)
	// }

	os.Exit(m.Run())
}

// Config contains the configuration values for the certifier tests
// This includes the gateway and auth parameters as well as the feature
// flags to control skipping specific tests.
type Config struct {
	// Gateway is the URL for the gateway that will be tested
	Gateway string
	// Auth contains the parsed proxy client auth
	Auth proxy.ClientAuth

	// SecretUpdate enables/disables the secret update test
	SecretUpdate bool
	// ScaleToZero enables/disables the scale from zero test
	ScaleToZero bool
}
