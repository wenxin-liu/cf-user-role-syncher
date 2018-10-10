package main

import (
	"google.golang.org/api/admin/directory/v1"
)

// Helper function for getRoleMembersDiff
func groupContainsMember(member string, groupMembers []*admin.Member) bool {
	for _, groupMember := range groupMembers {
		if groupMember.Email == member {
			return true
		}
	}
	return false
}
