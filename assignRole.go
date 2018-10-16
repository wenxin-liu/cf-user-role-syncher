package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
)

func assignRole(group *Group, username string) error {
	// Set http PUT payload
	var payload string = `{"username": "` + username + `"}`
	// Make sure the user is associated with the org. When setting an org role this is actually
	// not really necessary, but for setting space roles it is! If not, you'll receive an
	// "error_code": "CF-InvalidRelation", "code": 1002 when setting the space role
	resp := sendHttpRequest("PUT", os.Getenv(EnvCfApiEndPoint)+"/v2/organizations/"+group.CfOrgGuid+"/users", nil, payload)
	defer resp.Body.Close()
	if resp.StatusCode == 201 {
		log.Println("Successfully associated user '" + username + "' to org " + group.Org)
	} else {
		return errors.New("Failed to associated user '" + username + "' to org " + group.Org)
	}
	// Check if an Org Role or a Space Role needs to be assigned
	if group.Space != "" {
		// A Space Role needs to be assigned
		// Get the Space GUID
		q := url.Values{}
		q.Add("q", "name:"+group.Space)
		q.Add("q", "organization_guid:"+group.CfOrgGuid)
		// Send HTTP Request to CF API
		resp = sendHttpRequest("GET", os.Getenv(EnvCfApiEndPoint)+"/v2/spaces", &q, "")
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
		resp = sendHttpRequest("PUT", os.Getenv(EnvCfApiEndPoint)+"/v2/spaces/"+spaces.Resources[0].Metadata.GUID+roleMap[group.Role], nil, payload)
		defer resp.Body.Close()
		if resp.StatusCode == 201 {
			log.Println("Successfully assigned SpaceRole '" + group.Role + "' to member " + username)
		} else {
			return errors.New("Failed to assign SpaceRole '" + group.Role + "' to member " + username)
		}
	} else if group.Role == "spacedeveloper" || group.Role == "SpaceDeveloper" {
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

		//Lastly, for every space in the org, associate user as SpaceDeveloper using username
		for _, r := range AllSpacesFromAnOrg.Resources {
			resp := sendHttpRequest("PUT", os.Getenv(EnvCfApiEndPoint)+"/v2/spaces/"+r.Metadata.GUID+"/developers", nil, payload)
			defer resp.Body.Close()
			fmt.Println("Successfully associated " + username + " to space " + r.Entity.Name + " in org " + group.Org + " as SpaceDeveloper")
		}

	} else {
		// An Org Role needs to be assigned
		// Map for mapping role name to CF API resource path
		roleMap := map[string]string{
			"orgmanager":     "/managers",
			"billingmanager": "/billing_managers",
			"auditor":        "/auditors",
		}
		resp = sendHttpRequest("PUT", os.Getenv(EnvCfApiEndPoint)+"/v2/organizations/"+group.CfOrgGuid+roleMap[group.Role], nil, payload)
		defer resp.Body.Close()
		if resp.StatusCode == 201 {
			log.Println("Successfully assigned OrgRole '" + group.Role + "' to member " + username)
		} else {
			return errors.New("Failed to assign OrgRole '" + group.Role + "' to member " + username)
		}
	}
	// Role assignment was successful
	//fmt.Println(group.CfOrgGuid)
	return nil
}

