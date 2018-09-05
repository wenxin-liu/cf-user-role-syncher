package main

import (
    "os"
    "fmt"
    "log"
    "strings"
    "net/http"
    "net/url"
    "encoding/json"
    "strconv"
    "bytes"

    "golang.org/x/net/context"
    "google.golang.org/api/admin/directory/v1"
)

type Config struct {
    AccessToken string
    CFApiEndpoint string
}

type ApiResult struct {
    Resources    []struct {
        Metadata struct {
            GUID       string    `json:"guid"`
        } `json:"metadata"`
    } `json:"resources"`
}

var cliOptionsMsg = `Possible options:
- gmapper token

`
var confFile string = "config.json"
var tokFile string  = "token.json"
var credFile string = "credentials.json"
var config Config


func main() {
    // Check command line arguments
    if len(os.Args) > 1 {
        switch os.Args[1] {
        case "token":
            genOauthToken()
        default:
            fmt.Print(cliOptionsMsg)
        }
    } else {
        startMapper()
    }
}


func genOauthToken() {
    fmt.Println("Will generate file " + tokFile + " for Google Directory Admin API")
    // Load oauth.Config (e.g. Google oauth endpoint, client_id, client_secret)
    oauthConf := getOauthConfig(credFile)
    // Start oauth process on the web to get oauth token 
    err := getTokenFromWeb(oauthConf, tokFile)
    if err != nil {
        log.Fatalf("Unable to create oauth token: %v", err)
    } else {
        fmt.Println(tokFile + " created!")
    }
}


func startMapper() {
    // Load config
    configFromFile(&config, confFile)
    // Load oauth.Config (e.g. Google oauth endpoint)
    oauthConf := getOauthConfig(credFile)
    // Load existing oauth token (access_key and resfresh_key)
    oauthTok, err := tokenFromFile(tokFile)
    // Create 'Service' so Google Directory (Admin) can be requested
    httpClient := oauthConf.Client(context.Background(), oauthTok)
    googleService, err := admin.New(httpClient)
    if err != nil {
        log.Fatalf("Unable to retrieve directory Client: %v", err)
    }
    // Search for all Google Groups matching the search pattern
    groupsRes, err := googleService.Groups.List().Customer("my_customer").Query("email:snpaas__*").MaxResults(10).Do()
    if err != nil {
        log.Fatalf("Unable to retrieve Google Groups: %v", err)
    }
    if len(groupsRes.Groups) == 0 {
        fmt.Println("No groups found.\n")
    } else {
        for _, gr := range groupsRes.Groups {
            fmt.Printf("GROUP EMAIL: %s\n", gr.Email)

            membersRes, err := googleService.Members.List(gr.Email).MaxResults(10).Do()
            if err != nil {
                log.Fatalf("Unable to retrieve members in group: %v", err)
            }
            if len(membersRes.Members) == 0 {
                fmt.Println("No members found.\n")
            } else {
                fmt.Println("MEMBERS:")
                for _, m := range membersRes.Members {
                    fmt.Printf("%s\n", m.Email)
                    // Start process of assigning the right CF role based on group email address
                    assignRole(gr.Email, m.Email)
                }
            }
        } // End for
    } // End else
} // End startMapper


func assignRole(groupEmail string, username string) {
    // Get the part of the group email address before the '@'
    mailboxName := strings.Split(groupEmail, "@")[0]
    fmt.Println("Part before @: " + mailboxName)
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
        log.Println("Not a valid group email format! Role assignment fails for group: " + groupEmail)
        return
    }
    // First we need to get the org GUID
    // Set query string parameters to search org
    org = groupAttr[1]
    //fmt.Println("ORG = " + org)
    q := url.Values{}
    q.Add("q", "name:" + org)
    q.Add("inline-relations-depth", "1")
    // Send HTTP Request to CF API
    resp := sendHttpRequest("GET", config.CFApiEndpoint + "/v2/organizations", &q, "")
    // Callers should close resp.Body
    // when done reading from it
    // Defer the closing of the body
    defer resp.Body.Close()
    // Create new ApiResult data set and parse json from the response
    var orgs ApiResult
    if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
        log.Println(err)
    }
    // Check if there is exactly one org found
    fmt.Println("Array length: " + strconv.Itoa(len(orgs.Resources)))
    if len(orgs.Resources) != 1 {
        fmt.Println("Search for org '" + org + "' did not result in exactly 1 match!")
        return
    }
    fmt.Println("GUID = ", orgs.Resources[0].Metadata.GUID)
    // Set http PUT payload
    var payload string = `{"username": "` + username + `"}`
    // Check if an Org Role or a Space Role needs to be assigned
    if space != "" {
        // A Space Role needs to be assigned
        fmt.Println("Space is: " + space)
        // Get the Space GUID
        q := url.Values{}
        q.Add("q", "name:" + space)
        q.Add("q", "organization_guid:" + orgs.Resources[0].Metadata.GUID)
        // Send HTTP Request to CF API
        resp = sendHttpRequest("GET", config.CFApiEndpoint + "/v2/spaces", &q, "")
        defer resp.Body.Close()
        // Create new ApiResult data set and parse json from the response
        var spaces ApiResult
        if err := json.NewDecoder(resp.Body).Decode(&spaces); err != nil {
            log.Println(err)
        }
        if len(spaces.Resources) != 1 {
            fmt.Println("Search for space '" + space + "' did not result in exactly 1 match!")
            return
        }
        fmt.Println("Space ID: " + spaces.Resources[0].Metadata.GUID)
        // Map for mapping role name to CF API resource path
        roleMap := map[string]string{
            "spacemanager": "/managers",
            "spacedeveloper": "/developers",
            "spaceauditor": "/auditors",
        }
        resp = sendHttpRequest("PUT", config.CFApiEndpoint + "/v2/spaces/" + spaces.Resources[0].Metadata.GUID + roleMap[role], nil, payload)
        defer resp.Body.Close()
        if resp.StatusCode == 201 {
            fmt.Println("Succesfully assigned SpaceRole '" + role + "' to member " + username)
        } else {
            fmt.Println("Failed to assign SpaceRole '" + role + "' to member " + username)
        }
        fmt.Println("Status code: " + strconv.Itoa(resp.StatusCode))
    } else {
        // A Org Role needs to be assigned
        // Map for mapping role name to CF API resource path
        roleMap := map[string]string{
            "orgmanager": "/managers",
            "billingmanager": "/billing_managers",
            "auditor": "/auditors",
        }
        resp = sendHttpRequest("PUT", config.CFApiEndpoint + "/v2/organizations/" + orgs.Resources[0].Metadata.GUID + roleMap[role], nil, payload)
        defer resp.Body.Close()
        if resp.StatusCode == 201 {
            fmt.Println("Succesfully assigned OrgRole '" + role + "' to member " + username)
        } else {
            fmt.Println("Failed to assign OrgRole '" + role + "' to member " + username)
        }
        fmt.Println("Status code: " + strconv.Itoa(resp.StatusCode))
    }
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
    fmt.Println(req.URL.String())
    // Set Headers
    // TEMPORARILY WE USE METHOD getAccessTokenCF()
    req.Header.Add("Authorization", getAccessTokenCF())
    // Execute request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Fatal("Do: ", err)
    }
    return resp
}