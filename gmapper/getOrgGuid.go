package main

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
)

func getOrgGuid(org string) (string, error) {
	// Set query string parameters to search org
	q := url.Values{}
	q.Add("q", "name:"+org)
	q.Add("inline-relations-depth", "1")
	// Send HTTP Request to CF API
	resp := sendHttpRequest("GET", os.Getenv(EnvCfApiEndPoint)+"/v2/organizations", &q, "")
	// Callers should close resp.Body
	// when done reading from it
	// Defer the closing of the body
	defer resp.Body.Close()
	// Create new ApiResult data set and parse json from the response
	var orgs ApiResult
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return "", err
	}
	// Check if there is exactly one org found
	if len(orgs.Resources) != 1 {
		return "", errors.New("Search for org '" + org + "' did not result in exactly 1 match!")
	}
	return orgs.Resources[0].Metadata.GUID, nil
}
