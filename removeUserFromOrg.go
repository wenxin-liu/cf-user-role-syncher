package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"os"

	"github.com/SpringerPE/cf-user-role-syncher/token"
)

func removeUserFromOrg(group *Group, username string) error {
	type UserSummary struct {
		Entity struct {
			Organizations []struct {
				Metadata struct {
					GUID string `json:"guid"`
				} `json:"metadata"`
				Entity struct {
					Spaces []struct {
						Metadata struct {
							GUID string `json:"guid"`
						} `json:"metadata"`
					} `json:"spaces"`
				} `json:"entity"`
			} `json:"organizations"`
			ManagedOrganizations []struct {
				Metadata struct {
					GUID string `json:"guid"`
				} `json:"metadata"`
			} `json:"managed_organizations"`
			BillingManagedOrganizations []struct {
				Metadata struct {
					GUID string `json:"guid"`
				} `json:"metadata"`
			} `json:"billing_managed_organizations"`
			AuditedOrganizations []struct {
				Metadata struct {
					GUID string `json:"guid"`
				} `json:"metadata"`
			} `json:"audited_organizations"`
			Spaces []struct {
				Metadata struct {
					GUID string `json:"guid"`
				} `json:"metadata"`
			} `json:"spaces"`
			ManagedSpaces []struct {
				Metadata struct {
					GUID string `json:"guid"`
				} `json:"metadata"`
			} `json:"managed_spaces"`
			AuditedSpaces []struct {
				Metadata struct {
					GUID string `json:"guid"`
				} `json:"metadata"`
			} `json:"audited_spaces"`
		} `json:"entity"`
	} // End of struct
	//
	// First get the users GUID
	q := url.Values{}
	q.Add("attributes", "id")
	q.Add("filter", "userName eq \""+username+"\"")
	resp := sendHttpRequest("GET", os.Getenv(token.EnvUaaEndPoint)+"/Users", &q, "")
	defer resp.Body.Close()
	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return err
	}
	// we need exactly 1 resource to be returned
	if len(user.Resources) != 1 {
		return errors.New("Search for user '" + username + "' did not return exactly 1 resource!")
	}
	// GUID of user is user.Resources[0].ID
	// Get user summary which contains all the user's role memberships for orgs and spaces
	resp = sendHttpRequest("GET", os.Getenv(EnvCfApiEndPoint)+"/v2/users/"+user.Resources[0].ID+"/summary", nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("Failed to get user summary for user '" + username + "'")
	}
	// Create new UserSummary data set and parse json from the response
	var userSummary UserSummary
	if err := json.NewDecoder(resp.Body).Decode(&userSummary); err != nil {
		return err
	}
	// Is the user associated to at least one org?
	if !(len(userSummary.Entity.Organizations) >= 1) {
		// Nothing to do
		return nil
	}
	// Determine which org data matches the org we are removing the user from (group.CfOrgGuid)
	var orgIndex int
	var orgFound bool
	for i, org := range userSummary.Entity.Organizations {
		if org.Metadata.GUID == group.CfOrgGuid {
			orgIndex = i
			orgFound = true
			break
		}
	}
	// Nothing to do if the user is not associated to this org (group.CfOrgGuid) anymore
	if !orgFound {
		return nil
	}
	// The easiest check is if the user still has an org level permission for this org
	// If the user still has one, we can't remove the user from the org
	// Scan the Org Manager role memberships
	for _, guid := range userSummary.Entity.ManagedOrganizations {
		if guid.Metadata.GUID == group.CfOrgGuid {
			return nil
		}
	}
	// Scan the Billing Manager role memberships
	for _, guid := range userSummary.Entity.BillingManagedOrganizations {
		if guid.Metadata.GUID == group.CfOrgGuid {
			return nil
		}
	}
	// Scan the Auditor role memberships
	for _, guid := range userSummary.Entity.AuditedOrganizations {
		if guid.Metadata.GUID == group.CfOrgGuid {
			return nil
		}
	}
	// At this point its obvious the user as no org level roles within this org anymore.
	// We do still need to check space level roles
	// The space role membership arrays only contain the space GUID, not to what org it belongs
	// Therefore we'll iterate over all space role memberships and check for every
	// space GUID if they are part of the org
	// Scan the Space Developer role memberships
	for _, space := range userSummary.Entity.Spaces {
		// Check if the Space GUID is part of the org
		for _, orgSpace := range userSummary.Entity.Organizations[orgIndex].Entity.Spaces {
			if orgSpace.Metadata.GUID == space.Metadata.GUID {
				return nil
			}
		}
	}
	// Scan the Space Manager role memberships
	for _, space := range userSummary.Entity.ManagedSpaces {
		// Check if the Space GUID is part of the org
		for _, orgSpace := range userSummary.Entity.Organizations[orgIndex].Entity.Spaces {
			if orgSpace.Metadata.GUID == space.Metadata.GUID {
				return nil
			}
		}
	}
	// Scan the Space Auditor role memberships
	for _, space := range userSummary.Entity.AuditedSpaces {
		// Check if the Space GUID is part of the org
		for _, orgSpace := range userSummary.Entity.Organizations[orgIndex].Entity.Spaces {
			if orgSpace.Metadata.GUID == space.Metadata.GUID {
				return nil
			}
		}
	}
	// At this point we know the user has no org or space role in this org
	// We can remove the user from the org
	resp = sendHttpRequest("DELETE", os.Getenv(EnvCfApiEndPoint)+"/v2/organizations/"+group.CfOrgGuid+"/users/"+user.Resources[0].ID, nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return errors.New("Failed to remove '" + username + "' from org " + group.Org)
	} else {
		log.Println("Removing '" + username + "' from org was successful")
	}
	return nil
}
