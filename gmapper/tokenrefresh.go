package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	Jti          string `json:"jti"`
}

func getTokenFromUaa() []byte {
	body := strings.NewReader(`client_id=cf&client_secret=&grant_type=refresh_token&refresh_token=` + config.RefreshToken)
	req, err := http.NewRequest("POST", config.UaaApiEndpoint + "/oauth/token", body)
	if err != nil {
		// handle err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()

	body1, err := ioutil.ReadAll(resp.Body)

	return body1
}

func UnmarshalJson(tokenFromUaa []byte) (string) {
	var tokenresponse TokenResponse
	json.Unmarshal(tokenFromUaa, &tokenresponse)

	//fmt.Printf("Access Token: %s, Refresh Token: %s", tokenresponse.AccessToken, tokenresponse.RefreshToken)

	a := tokenresponse.AccessToken
	//b := tokenresponse.RefreshToken
	s := "bearer " + a
	return s
	//return ""
}