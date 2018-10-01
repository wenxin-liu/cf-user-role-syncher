package main

import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "strings"
    
    "golang.org/x/net/context"
    "google.golang.org/api/admin/directory/v1"
    "github.com/SpringerPE/cf-user-role-syncher/gmapper/token"
)

// Declaration of environment variable key names
const EnvCfApiEndPoint string = "CFAPIENDPOINT"

// Structure for getting GUID of Orgs and Spaces in CF
type ApiResult struct {
    Resources    []struct {
        Metadata struct {
            GUID       string    `json:"guid"`
        } `json:"metadata"`
    } `json:"resources"`
}

// Stucture for getting members of a CF Role
type RoleMembers struct {
    Resources    []struct {
        Metadata struct {
            GUID      string    `json:"guid"`
        } `json:"metadata"`
        Entity struct {
            Username  string     `json:"username"`
        } `json:"entity"`
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
    TotalResults int      `json:"totalResults"`
}

// Structure which holds the GUID of a user
// The GUID should be returned when new user is created in UAA
type UaaGuid struct {
    ID   string `json:"id"`
}

// Will hold info for every individual group
// as every group represent a single combination of Org, Space and Role.
type Group struct {
    Org         string
    Space       string
    Role        string
    CfOrgGuid   string
}

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
        log.Fatalf("Unable to retrieve Google Groups: %v", err)  // Exit program
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
                        log.Printf("Could not create new user in CF/UAA for user '" + m.Email +"': %v\n", err)
                        continue // Try next member
                    }
                    // Start process of assigning the right CF Org/Space role to this member
                    if err := assignRole(group, m.Email); err != nil {
                        log.Printf("Could not assign role for user '" + m.Email + "': %v\n", err)
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
                    log.Printf("Could not unset role for user '" + username + "': %v\n", err)
                    continue // Try to unset role for next user
                }
                if err := removeUserFromOrg(group, username); err != nil {
                    log.Printf("Could not remove user '" + username + "' from org: %v\n", err)
                    continue // Try to unset role for next user
                }
            }
        } // End for (groups)
    } // End else
} // End startMapper


func scrapeGroupAttributes(email string) (*Group, error) {
    // Get the part of the group email address before the '@'
    mailboxName := strings.Split(email, "@")[0]
    // Split the mailboxName to get org, space and role
    groupAttr := strings.Split(mailboxName, "__")
    var org, space, role string
    // 3 items in group email = Org role
    // 4 items in group email = Space role
    if len(groupAttr) == 3 {
        role  = groupAttr[2]
    } else if len(groupAttr) == 4 {
        space = groupAttr[2]
        role  = groupAttr[3]
    } else {
        return nil, errors.New("Not a valid group email format for email: " + email)
    }
    org = groupAttr[1]
    var group = Group{
        Org: org,
        Space: space,
        Role: role,
    }
    return &group, nil
}


func getOrgGuid(org string) (string, error) {
    // Set query string parameters to search org
    q := url.Values{}
    q.Add("q", "name:" + org)
    q.Add("inline-relations-depth", "1")
    // Send HTTP Request to CF API
    resp := sendHttpRequest("GET", os.Getenv(EnvCfApiEndPoint) + "/v2/organizations", &q, "")
    // Callers should close resp.Body
    // when done reading from it
    // Defer the closing of the body
    defer resp.Body.Close()
    // Create new ApiResult data set and parse json from the response
    var orgs ApiResult
    if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
        return "", err
    }
    // Check if there is exactly one org found
    if len(orgs.Resources) != 1 {
        return "", errors.New("Search for org '" + org + "' did not result in exactly 1 match!")
    }
    return orgs.Resources[0].Metadata.GUID, nil
}


// Will create a new user in CF/UAA
// The user gets an 'origin' set to the SSO provider name
// If the user account already exists, nothing will be done here.
func createShadowUserCF(username string) error {
    // Search uaa to check if the username exists
    // attributes=id,externalId,userName,active,origin,lastLogonTime
    // filter=userName eq "gerard.laan@springernature.com"
    q := url.Values{}
    q.Add("attributes", "id,externalId,userName,active,origin,lastLogonTime")
    q.Add("filter", "userName eq \"" + username + "\"")
    resp := sendHttpRequest("GET", os.Getenv(token.EnvUaaEndPoint) + "/Users", &q, "")
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
        resp := sendHttpRequest("POST", os.Getenv(token.EnvUaaEndPoint) + "/Users", nil, payload)
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
        resp = sendHttpRequest("POST", os.Getenv(EnvCfApiEndPoint) + "/v2/users", nil, payload)
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


func assignRole(group *Group, username string) error {    
    // Set http PUT payload
    var payload string = `{"username": "` + username + `"}`
    // Make sure the user is associated with the org. When setting an org role this is actually
    // not really necessary, but for setting space roles it is! If not, you'll receive an 
    // "error_code": "CF-InvalidRelation", "code": 1002 when setting the space role
    resp := sendHttpRequest("PUT", os.Getenv(EnvCfApiEndPoint) + "/v2/organizations/" + group.CfOrgGuid + "/users", nil, payload)
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
        q.Add("q", "name:" + group.Space)
        q.Add("q", "organization_guid:" + group.CfOrgGuid)
        // Send HTTP Request to CF API
        resp = sendHttpRequest("GET", os.Getenv(EnvCfApiEndPoint) + "/v2/spaces", &q, "")
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
            "spacemanager": "/managers",
            "spacedeveloper": "/developers",
            "spaceauditor": "/auditors",
        }
        resp = sendHttpRequest("PUT", os.Getenv(EnvCfApiEndPoint) + "/v2/spaces/" + spaces.Resources[0].Metadata.GUID + roleMap[group.Role], nil, payload)
        defer resp.Body.Close()
        if resp.StatusCode == 201 {
            log.Println("Successfully assigned SpaceRole '" + group.Role + "' to member " + username)
        } else {
            return errors.New("Failed to assign SpaceRole '" + group.Role + "' to member " + username)
        }
    } else {
        // An Org Role needs to be assigned
        // Map for mapping role name to CF API resource path
        roleMap := map[string]string{
            "orgmanager": "/managers",
            "billingmanager": "/billing_managers",
            "auditor": "/auditors",
        }
        resp = sendHttpRequest("PUT", os.Getenv(EnvCfApiEndPoint) + "/v2/organizations/" + group.CfOrgGuid + roleMap[group.Role], nil, payload)
        defer resp.Body.Close()
        if resp.StatusCode == 201 {
            log.Println("Successfully assigned OrgRole '" + group.Role + "' to member " + username)
        } else {
            return errors.New("Failed to assign OrgRole '" + group.Role + "' to member " + username)
        }
    }
    // Role assignment was successful
    return nil
}


func getCfRoleMembers(group *Group) ([]string, error) {
    var roleMembers []string
    var members RoleMembers
    // Check if an Org Role or a Space Role needs to be unset
    if group.Space != "" {
        // A Space Role needs to be unset
        // Get the Space GUID
        q := url.Values{}
        q.Add("q", "name:" + group.Space)
        q.Add("q", "organization_guid:" + group.CfOrgGuid)
        // Send HTTP Request to CF API
        resp := sendHttpRequest("GET", os.Getenv(EnvCfApiEndPoint) + "/v2/spaces", &q, "")
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
            "spacemanager": "/managers",
            "spacedeveloper": "/developers",
            "spaceauditor": "/auditors",
        }
        resp = sendHttpRequest("GET", os.Getenv(EnvCfApiEndPoint) + "/v2/spaces/" + spaces.Resources[0].Metadata.GUID + roleMap[group.Role], nil, "")
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
            "orgmanager": "/managers",
            "billingmanager": "/billing_managers",
            "auditor": "/auditors",
        }
        resp := sendHttpRequest("GET", os.Getenv(EnvCfApiEndPoint) + "/v2/organizations/" + group.CfOrgGuid + roleMap[group.Role], nil, "")
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
        resp := sendHttpRequest("GET", os.Getenv(token.EnvUaaEndPoint) + "/Users/" + member.Metadata.GUID , nil, "")
        defer resp.Body.Close()
        if resp.StatusCode != 200 {
            return roleMembers, errors.New("Failed to check UAA for the origin of user '" + member.Entity.Username + "'.")
        }
        type UaaUser struct {
            Origin  string  `json:"origin"`
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
// Helper function for getRoleMembersDiff
func groupContainsMember(member string, groupMembers []*admin.Member) bool {
    for _, groupMember := range groupMembers {
        if groupMember.Email == member {
            return true
        }
    }
    return false
}


func unsetRole(group *Group, username string) error {
    // Set http PUT payload
    var payload string = `{"username": "` + username + `"}`
    // Check if an Org Role or a Space Role needs to be unset
    if group.Space != "" {
        // A Space Role needs to be unset
        // Get the Space GUID
        q := url.Values{}
        q.Add("q", "name:" + group.Space)
        q.Add("q", "organization_guid:" + group.CfOrgGuid)
        // Send HTTP Request to CF API
        resp := sendHttpRequest("GET", os.Getenv(EnvCfApiEndPoint) + "/v2/spaces", &q, "")
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
            "spacemanager": "/managers",
            "spacedeveloper": "/developers",
            "spaceauditor": "/auditors",
        }
        resp = sendHttpRequest("POST", os.Getenv(EnvCfApiEndPoint) + "/v2/spaces/" + spaces.Resources[0].Metadata.GUID + roleMap[group.Role] + "/remove", nil, payload)
        defer resp.Body.Close()
        if resp.StatusCode != 200 {
            return errors.New("Failed to unset role '" + group.Role + "' for member " + username)
        }
    } else {
        // An Org Role needs to be unset
        // Map for mapping role name to CF API resource path
        roleMap := map[string]string{
            "orgmanager": "/managers",
            "billingmanager": "/billing_managers",
            "auditor": "/auditors",
        }
        resp := sendHttpRequest("POST", os.Getenv(EnvCfApiEndPoint) + "/v2/organizations/" + group.CfOrgGuid + roleMap[group.Role] + "/remove", nil, payload)
        defer resp.Body.Close()
        if resp.StatusCode != 204 {
            return errors.New("Failed to unset role '" + group.Role + "' for member " + username)
        }
    }
    // Unset role was successful
    log.Println("Unset role '" + group.Role + "' for user '" + username + "' was successful")
    return nil
}


func removeUserFromOrg(group *Group, username string) error {
    type UserSummary struct {
        Entity struct {
            Organizations []struct {
                Metadata struct {
                    GUID      string    `json:"guid"`
                } `json:"metadata"`
                Entity struct {
                    Spaces         []struct {
                        Metadata struct {
                            GUID      string    `json:"guid"`
                        } `json:"metadata"`
                    } `json:"spaces"`
                } `json:"entity"`
            } `json:"organizations"`
            ManagedOrganizations []struct {
                Metadata struct {
                    GUID      string    `json:"guid"`
                } `json:"metadata"`
            } `json:"managed_organizations"`
            BillingManagedOrganizations []struct {
                Metadata struct {
                    GUID      string    `json:"guid"`
                } `json:"metadata"`
            } `json:"billing_managed_organizations"`
            AuditedOrganizations []struct {
                Metadata struct {
                    GUID      string    `json:"guid"`
                } `json:"metadata"`
            } `json:"audited_organizations"`
            Spaces []struct {
                Metadata struct {
                    GUID      string    `json:"guid"`
                } `json:"metadata"`
            } `json:"spaces"`
            ManagedSpaces []struct {
                Metadata struct {
                    GUID      string    `json:"guid"`
                } `json:"metadata"`
            } `json:"managed_spaces"`
            AuditedSpaces []struct {
                Metadata struct {
                    GUID      string    `json:"guid"`
                } `json:"metadata"`
            } `json:"audited_spaces"`
        } `json:"entity"`
    } // End of struct
    //
    // First get the users GUID
    q := url.Values{}
    q.Add("attributes", "id")
    q.Add("filter", "userName eq \"" + username + "\"")
    resp := sendHttpRequest("GET", os.Getenv(token.EnvUaaEndPoint) + "/Users", &q, "")
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
    resp = sendHttpRequest("GET", os.Getenv(EnvCfApiEndPoint) + "/v2/users/" + user.Resources[0].ID + "/summary", nil, "")
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
    resp = sendHttpRequest("DELETE", os.Getenv(EnvCfApiEndPoint) + "/v2/organizations/" + group.CfOrgGuid + "/users/" + user.Resources[0].ID, nil, "")
    defer resp.Body.Close()
    if resp.StatusCode != 204 {
        return errors.New("Failed to remove '" + username + "' from org " + group.Org)
    } else {
        log.Println("Removing '" + username + "' from org was successful")
    }
    return nil
}


func sendHttpRequest(method string, url string, querystring *url.Values, payload string) *http.Response {
    // Create new http request
    req, err := http.NewRequest(method, url, bytes.NewBufferString(payload))
    if err != nil {
        log.Print(err)
        os.Exit(1)
    }
    // Check if any query string parameters are supplied. If yes, add them to the request
    if querystring != nil {
        req.URL.RawQuery = querystring.Encode()
    }
    //fmt.Println(req.URL.String())
    // Set Headers
    uaaresponse := token.GetTokenFromUaa()
    req.Header.Add("Authorization", token.UnmarshalJson(uaaresponse))
    if (method == "POST") || (method == "PUT") {
        req.Header.Add("Content-Type", "application/json")
    }
    // Execute request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Fatal("Do: ", err)
    }
    // In case the response is not HTTP 2xx (success), we would like to know what 
    // is in the response body. (Most likely some error which could be helpful)
    if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
        // Convert response body into a string
        bodyBytes, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            log.Println("Not able to output the error for unsuccessful HTTP request (no 2xx code)")
        } else {
            bodyString := string(bodyBytes)
            // Log multi line string
            log.Println("Error while sending HTTP request:\n" +
                "HTTP status code: " + strconv.Itoa(resp.StatusCode) + "\n" +
                "HTTP response body:" + bodyString)
        }
    } // End if (when http response is not 2xx)
    // All done processing the http request. Return the response instance
    return resp
}