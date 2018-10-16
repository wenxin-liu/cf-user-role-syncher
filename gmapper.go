package main

import (
	"fmt"
	"os"

	"github.com/SpringerPE/cf-user-role-syncher/token"
)

// Declaration of environment variable key names
const EnvCfApiEndPoint string = "CFAPIENDPOINT"

// ApiResult Structure for getting GUID of Orgs and Spaces in CF
type ApiResult struct {
	Resources []struct {
		Metadata struct {
			GUID string `json:"guid"`
		} `json:"metadata"`
	} `json:"resources"`
}

// Stucture for getting members of a CF Role
type RoleMembers struct {
	Resources []struct {
		Metadata struct {
			GUID string `json:"guid"`
		} `json:"metadata"`
		Entity struct {
			Username string `json:"username"`
		} `json:"entity"`
	} `json:"resources"`
}

// Structure for getting name of spaces
type Spaces struct {
	Resources []struct {
		Metadata struct {
			GUID string `json:"guid"`
		} `json:"metadata"`
		Entity struct {
			Name string `json:"name""`
		} `json:"entity""`
	} `json:"resources"`
}

// Structure for user details
// Used when searching on existence of user in UAA
type User struct {
	Resources []struct {
		LastLogonTime int64  `json:"lastLogonTime"`
		Origin        string `json:"origin"`
		ExternalID    string `json:"externalId"`
		Active        bool   `json:"active"`
		ID            string `json:"id"`
		UserName      string `json:"userName"`
	} `json:"resources"`
	TotalResults int `json:"totalResults"`
}

// Structure which holds the GUID of a user
// The GUID should be returned when new user is created in UAA
type UaaGuid struct {
	ID string `json:"id"`
}

// Will hold info for every individual group
// as every group represent a single combination of Org, Space and Role.
type Group struct {
	Org       string
	Space     string
	Role      string
	CfOrgGuid string
}

// This var holds the Oauth Access Token for CF
// Initializing this with a value similar to 'bearer something' is important
// This will make CF recognize the Access Token is invalid with the first request to CF
var cfAccessToken string = "bearer none"

// This message will show when not providing the right cli options
var cliOptionsMsg = `Possible options:
- gmapper token

`

func main() {
	// Check command line arguments
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "token":
			token.GenGoogleOauthToken()
		default:
			fmt.Print(cliOptionsMsg)
		}
	} else {
		startMapper()
	}
}
