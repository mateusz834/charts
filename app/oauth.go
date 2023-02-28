package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type OAuth struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
}

func (o *OAuth) getAccessToken(authorizatonCode string) (string, error) {
	body := make(url.Values)
	body.Add("grant_type", "authorization_code")
	body.Add("client_id", o.ClientID)
	body.Add("client_secret", o.ClientSecret)
	body.Add("code", authorizatonCode)

	req, err := http.NewRequest(http.MethodPost, o.TokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed while preparing access token request: %v", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed while sending access token request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed while retreiving access token request, got status code: %v", response.Status)
	}

	resBody := struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`

		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		ErrorURI         string `json:"error_uri"`
	}{}

	if err := json.NewDecoder(response.Body).Decode(&resBody); err != nil {
		return "", fmt.Errorf("failed while json unmarshaling: %v", err)
	}

	// rfc 6749 says that errors should be returned with 400 code, but github sends it with 200.
	// The authorization server responds with an HTTP 400 (Bad Request)
	// status code (unless specified otherwise) and includes the following
	// parameters with the response:
	if len(resBody.Error) != 0 {
		return "", fmt.Errorf("failed while receiving access token: %v: %v, more details: %v", resBody.Error, resBody.ErrorDescription, resBody.ErrorURI)
	}

	if resBody.TokenType != "bearer" {
		return "", fmt.Errorf("got non-bearer token type: %v", resBody.TokenType)
	}

	return resBody.AccessToken, nil
}
