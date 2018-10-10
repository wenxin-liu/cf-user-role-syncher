package main

import (
	"log"

	"github.com/SpringerPE/cf-user-role-syncher/gmapper/token"
	"golang.org/x/net/context"
	"google.golang.org/api/admin/directory/v1"
)

func startMapper() {
	// Load oauth.Config (e.g. Google oauth endpoint)
	oauthConf := token.GetOauthConfig()
	// Load oauth.Token for Google (e.g RefreshToken)
	oauthTok := token.GetOauthToken()
	// Create 'Service' so Google Directory (Admin) can be requested
	httpClient := oauthConf.Client(context.Background(), oauthTok)
	googleService, err := admin.New(httpClient)
	if err != nil {
		log.Fatalf("Unable to create new Google Service (Google client) instance: %v", err)
	}
	// Search for all Google Groups matching the search pattern
	groupsRes, err := googleService.Groups.List().Customer("my_customer").Query("email:snpaas__*").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve Google Groups: %v", err) // Exit program
	}
	if len(groupsRes.Groups) == 0 {
		log.Fatalln("No groups found.")
	} else {
		// Loop over all found groups
		for _, gr := range groupsRes.Groups {
			log.Printf("GROUP EMAIL: %v\n", gr.Email)
			// Get group attributes
			group, err := scrapeGroupAttributes(gr.Email)
			if err != nil {
				log.Printf("Could not scrape group attributes: %v\n", err)
				continue // Try next group
			}
			// Get Org GUID from CF
			group.CfOrgGuid, err = getOrgGuid(group.Org)
			if err != nil {
				log.Printf("Could not get Org GUID: %v\n", err)
				continue // Try next group
			}
			// Search members within this group
			groupMembersRes, err := googleService.Members.List(gr.Email).Do()
			if err != nil {
				log.Fatalf("Unable to retrieve members in group: %v", err) // Exit program
			}
			if len(groupMembersRes.Members) == 0 {
				log.Println("No members found.")
			} else {
				// Loop over all found members within this one group
				for _, m := range groupMembersRes.Members {
					// First make sure the username exists on CF/UAA side
					if err := createShadowUserCF(m.Email); err != nil {
						log.Printf("Could not create new user in CF/UAA for user '"+m.Email+"': %v\n", err)
						continue // Try next member
					}
					// Start process of assigning the right CF Org/Space role to this member
					if err := assignRole(group, m.Email); err != nil {
						log.Printf("Could not assign role for user '"+m.Email+"': %v\n", err)
						continue // Try next member
					}
				} // End for (members)
			} // End if (members)
			//
			// Unset the role for users who are not member of the group anymore
			// Get the role members in CF (so we can compare with the group members)
			roleMembers, err := getCfRoleMembers(group)
			if err != nil {
				log.Printf("Could not get list of existing role members from CF: %v\n", err)
				continue // Try next group
			}
			// Get a list of usernames which need the role to be unset for
			// (essentially the diff between the group members and role members in CF)
			unauthorizedUsers := getRoleMembersDiff(roleMembers, groupMembersRes.Members)
			// Unset the role for every user in the unauthorizedUsers list
			// And try to remove the user from the org when it doesn't have any role anymore
			for _, username := range unauthorizedUsers {
				if err := unsetRole(group, username); err != nil {
					log.Printf("Could not unset role for user '"+username+"': %v\n", err)
					continue // Try to unset role for next user
				}
				if err := removeUserFromOrg(group, username); err != nil {
					log.Printf("Could not remove user '"+username+"' from org: %v\n", err)
					continue // Try to unset role for next user
				}
			}
		} // End for (groups)
	} // End else
	//unmarshalJson(listAllSpacesInAnOrg("engineering-enablement"))
} // End startMapper
