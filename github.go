package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type githubUser struct {
	Login      string `json:"login"`
	ID         uint64 `json:"id"`
	ProfileURL string `json:"html_url"`
}

func getGithubUserData(accessToken string) (*githubUser, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed while creting github user data request: %v", err)
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed while sending github user data request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed while retreiving github user data, got status: %v", response.Status)
	}

	var user githubUser
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed while json unmarshaling: %v", err)
	}

	return &user, nil
}
