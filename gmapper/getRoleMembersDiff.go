package main

import (
	"google.golang.org/api/admin/directory/v1"
)

func getRoleMembersDiff(roleMembers []string, groupMembers []*admin.Member) []string {
	var unauthorizedUsers []string
	// Loop through all roleMembers and check if they are still member of the group
	for _, roleMember := range roleMembers {
		// If the roleMember does NOT exist in group, add the user to unauthorizedUsers
		if !groupContainsMember(roleMember, groupMembers) {
			unauthorizedUsers = append(unauthorizedUsers, roleMember)
		}
	}
	return unauthorizedUsers
}
