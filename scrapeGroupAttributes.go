package main

import (
	"errors"
	"strings"
)

func scrapeGroupAttributes(email string) (*Group, error) {
	// Get the part of the group email address before the '@'
	mailboxName := strings.Split(email, "@")[0]
	// Split the mailboxName to get org, space and role
	groupAttr := strings.Split(mailboxName, "__")
	var org, space, role string
	// 3 items in group email = Org role
	// 4 items in group email = Space role
	if len(groupAttr) == 3 {
		role = groupAttr[2]
	} else if len(groupAttr) == 4 {
		space = groupAttr[2]
		role = groupAttr[3]
	} else {
		return nil, errors.New("Not a valid group email format for email: " + email)
	}
	org = groupAttr[1]
	var group = Group{
		Org:   org,
		Space: space,
		Role:  role,
	}
	//fmt.Println(&group)
	return &group, nil
}
