package main

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"

	"github.com/SpringerPE/cf-user-role-syncher/gmapper/token"
)

func getCfRoleMembers(group *Group) ([]string, error) {
	var roleMembers []string
	var members RoleMembers
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
			return roleMembers, err
		}
		if len(spaces.Resources) != 1 {
			return roleMembers, errors.New("Search for space '" + group.Space + "' did not result in exactly 1 match!")
		}
		// Map for mapping role name to CF API resource path
		roleMap := map[string]string{
			"spacemanager":   "/managers",
			"spacedeveloper": "/developers",
			"spaceauditor":   "/auditors",
		}
		resp = sendHttpRequest("GET", os.Getenv(EnvCfApiEndPoint)+"/v2/spaces/"+spaces.Resources[0].Metadata.GUID+roleMap[group.Role], nil, "")
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return roleMembers, errors.New("Failed to get role members from CF.")
		} else {
			// Parse json from the response into RoleMembers data structure
			if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
				return roleMembers, err
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
		resp := sendHttpRequest("GET", os.Getenv(EnvCfApiEndPoint)+"/v2/organizations/"+group.CfOrgGuid+roleMap[group.Role], nil, "")
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return roleMembers, errors.New("Failed to get role members from CF.")
		} else {
			// Parse json from the response into RoleMembers data structure
			if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
				return roleMembers, err
			}
		}
	}
	// Loop through the API result (available through the RoleMembers data structure)
	for _, member := range members.Resources {
		// First check in UAA if the user was created as a SSO user
		// We only take those 'SSO users' into account
		resp := sendHttpRequest("GET", os.Getenv(token.EnvUaaEndPoint)+"/Users/"+member.Metadata.GUID, nil, "")
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return roleMembers, errors.New("Failed to check UAA for the origin of user '" + member.Entity.Username + "'.")
		}
		type UaaUser struct {
			Origin string `json:"origin"`
		}
		var uaaUser UaaUser
		// Parse json from the response into UaaUser data structure
		if err := json.NewDecoder(resp.Body).Decode(&uaaUser); err != nil {
			return roleMembers, err
		}
		// This is where we match the origin of the user
		if uaaUser.Origin == os.Getenv(token.EnvUaaSsoProvider) {
			// Add username to the roleMembers string array
			roleMembers = append(roleMembers, member.Entity.Username)
		}
	} // End for loop (through all role members)
	return roleMembers, nil
}
