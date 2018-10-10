package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"os"

	"github.com/SpringerPE/cf-user-role-syncher/gmapper/token"
)

// Will create a new user in CF/UAA
// The user gets an 'origin' set to the SSO provider name
// If the user account already exists, nothing will be done here.
func createShadowUserCF(username string) error {
	// Search uaa to check if the username exists
	// attributes=id,externalId,userName,active,origin,lastLogonTime
	// filter=userName eq "gerard.laan@springernature.com"
	q := url.Values{}
	q.Add("attributes", "id,externalId,userName,active,origin,lastLogonTime")
	q.Add("filter", "userName eq \""+username+"\"")
	resp := sendHttpRequest("GET", os.Getenv(token.EnvUaaEndPoint)+"/Users", &q, "")
	defer resp.Body.Close()
	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return err
	}
	// No user or 1 user is fine. More than 1 user in the search result is not okay!
	if len(user.Resources) > 1 {
		return errors.New("Search for user '" + username + "' resulted in more than 1 results!")
	} else if len(user.Resources) == 0 {
		// User not found, so this username needs to be created
		log.Println("User '" + username + "' does not exist. Will now be created.")
		// Set http PUT payload for sending to uaa
		var payload string = `{
            "emails": [
                {
                    "primary": true,
                    "value": "` + username + `"
                }
            ],
            "name": {
                "familyName": "` + username + `",
                "givenName": "` + username + `"
            },
            "origin": "` + os.Getenv(token.EnvUaaSsoProvider) + `",
            "userName": "` + username + `"
        }`
		// Send http request
		resp := sendHttpRequest("POST", os.Getenv(token.EnvUaaEndPoint)+"/Users", nil, payload)
		defer resp.Body.Close()
		if resp.StatusCode == 201 {
			log.Println("Successfully created user '" + username + "' in UAA")
		} else {
			return errors.New("Failed to created user '" + username + "' in UAA")
		}
		// Check if GUID is returned
		var guid UaaGuid
		if err := json.NewDecoder(resp.Body).Decode(&guid); err != nil {
			return err
		}
		// When the user was created in UAA above, the API response body should contain GUID for user
		if guid.ID == "" {
			return errors.New("GUID was empty in UAA Api call for user " + username)
		}
		// Set GUID in CF
		payload = `{"guid": "` + guid.ID + `"}`
		resp = sendHttpRequest("POST", os.Getenv(EnvCfApiEndPoint)+"/v2/users", nil, payload)
		defer resp.Body.Close()
		if resp.StatusCode == 201 {
			log.Println("Successfully set GUID for '" + username + "' in CF")
		} else {
			return errors.New("Failed to set GUID for '" + username + "' in CF")
		}
	}
	// When user already exists or user was successfully created, there is no error to return
	return nil
}
