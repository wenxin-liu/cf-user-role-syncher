package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
)

func unsetRole(group *Group, username string) error {
	// Set http PUT payload
	var payload string = `{"username": "` + username + `"}`
	// Check if an Org Role or a Space Role needs to be unset
	if group.Space != "" {
		// A Space Role needs to be unset
		// Get the Space GUID
		q := url.Values{}
		q.Add("q", "name:"+group.Space)
		q.Add("q", "organization_guid:"+group.CfOrgGuid)
		// Send HTTP Request to CF API
		resp := sendHttpRequest("GET", os.Getenv(EnvCfApiEndPoint)+"/v2/spaces", &q, "")
		defer resp.Body.Close()
		// Create new ApiResult data set and parse json from the response
		var spaces ApiResult
		if err := json.NewDecoder(resp.Body).Decode(&spaces); err != nil {
			return err
		}
		if len(spaces.Resources) != 1 {
			return errors.New("Search for space '" + group.Space + "' did not result in exactly 1 match!")
		}
		// Map for mapping role name to CF API resource path
		roleMap := map[string]string{
			"spacemanager":   "/managers",
			"spacedeveloper": "/developers",
			"spaceauditor":   "/auditors",
		}
		resp = sendHttpRequest("POST", os.Getenv(EnvCfApiEndPoint)+"/v2/spaces/"+spaces.Resources[0].Metadata.GUID+roleMap[group.Role]+"/remove", nil, payload)
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return errors.New("Failed to unset role '" + group.Role + "' for member " + username)
		}
	} else if group.Role=="spacedeveloper" || group.Role=="SpaceDeveloper" {
		//For members of Google Groups named prefix_org_spacerole, assign SpaceDeveloper role for every space in the org
		//First, sending request to api to list all the spaces in an org
		resp := sendHttpRequest("GET", os.Getenv(EnvCfApiEndPoint)+"/v2/organizations/"+group.CfOrgGuid+
			"/spaces", nil, "")
		defer resp.Body.Close()

		//Then, taking the response and storing only space GUIDs and space names from the org in var AllSpacesFromAnOrg
		var AllSpacesFromAnOrg Spaces
		if err := json.NewDecoder(resp.Body).Decode(&AllSpacesFromAnOrg); err != nil {
			return err
		}

		//Lastly, for every space in the org, unset user as SpaceDeveloper using username
		for _, r := range AllSpacesFromAnOrg.Resources {
			resp := sendHttpRequest("POST", os.Getenv(EnvCfApiEndPoint)+"/v2/spaces/"+r.Metadata.GUID+"/developers/remove", nil, payload)
			defer resp.Body.Close()
			fmt.Println("Successfully unset role spacedeveloper " + username + " to space " + r.Entity.Name + " in org " + group.Org)

			if resp.StatusCode != 200 {
				return errors.New("Failed to unset role spacedeveloper for member " + username + " in space " + r.Entity.Name + " in org " + group.Org)
			}
		}

	} else {
		// An Org Role needs to be unset
		// Map for mapping role name to CF API resource path
		roleMap := map[string]string{
			"orgmanager":     "/managers",
			"billingmanager": "/billing_managers",
			"auditor":        "/auditors",
		}
		resp := sendHttpRequest("POST", os.Getenv(EnvCfApiEndPoint)+"/v2/organizations/"+group.CfOrgGuid+roleMap[group.Role]+"/remove", nil, payload)
		defer resp.Body.Close()
		if resp.StatusCode != 204 {
			return errors.New("Failed to unset role '" + group.Role + "' for member " + username)
		}
	}
	// Unset role was successful
	log.Println("Unset role '" + group.Role + "' for user '" + username + "' was successful")
	return nil
}
