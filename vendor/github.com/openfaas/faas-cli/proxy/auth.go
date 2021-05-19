package proxy

import (
	"net/http"

	"github.com/openfaas/faas-cli/config"
)

//CLIAuth auth struct for the CLI
type CLIAuth struct {
	Username string
	Password string
	Token    string
}

//BasicAuth basic authentication type
type BasicAuth struct {
	username string
	password string
}

func (auth *BasicAuth) Set(req *http.Request) error {
	req.SetBasicAuth(auth.username, auth.password)
	return nil
}

//BearerToken bearer token
type BearerToken struct {
	token string
}

func (c *BearerToken) Set(req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+c.token)
	return nil
}

//NewCLIAuth returns a new CLI Auth
func NewCLIAuth(token string, gateway string) (ClientAuth, error) {
	authConfig, _ := config.LookupAuthConfig(gateway)

	var (
		username    string
		password    string
		bearerToken string
		err         error
	)

	if authConfig.Auth == config.BasicAuthType {
		username, password, err = config.DecodeAuth(authConfig.Token)
		if err != nil {
			return nil, err
		}

		return &BasicAuth{
			username: username,
			password: password,
		}, nil

	}

	// User specified token gets priority
	if len(token) > 0 {
		bearerToken = token
	} else {
		bearerToken = authConfig.Token
	}

	return &BearerToken{
		token: bearerToken,
	}, nil
}
