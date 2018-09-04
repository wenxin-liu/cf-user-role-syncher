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
    // "bytes"

    "golang.org/x/net/context"
    "google.golang.org/api/admin/directory/v1"
)

type Config struct {
    AccessToken string
    CFApiEndpoint string
}

type Organizations struct {
    Resources    []struct {
        Metadata struct {
            GUID       string    `json:"guid"`
            URL        string    `json:"url"`
        } `json:"metadata"`
        Entity struct {
            Name      string      `json:"name"`
            Spaces    []struct {
                Metadata struct {
                    GUID      string    `json:"guid"`
                    URL       string    `json:"url"`
                } `json:"metadata"`
                Entity struct {
                    Name                     string      `json:"name"`
                } `json:"entity"`
            } `json:"spaces"`
        } `json:"entity"`
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
    // Split the group email address to get org, space and role
    roleMap := strings.Split(groupEmail, "__")
    var org, space, role string
    // 3 items in group email = Org role
    // 4 items in group email = Space role
    if len(roleMap) == 3 {
        role  = roleMap[2]
    } else if len(roleMap) == 4 {
        space = roleMap[2]
        role  = roleMap[3]
    } else {
        log.Println("Not a valid group email format! Role assignment fails for group: " + groupEmail)
        return
    }
    // First we need to get the org GUID
    // Set query string parameters to search org
    org = roleMap[1]
    //fmt.Println("ORG = " + org)
    q := url.Values{}
    q.Add("q", "name:" + org)
    q.Add("inline-relations-depth", "1")
    // Send HTTP Request to CF API
    resp := sendHttpRequest("GET", config.CFApiEndpoint + "/v2/organizations", &q)
    // Callers should close resp.Body
    // when done reading from it
    // Defer the closing of the body
    defer resp.Body.Close()
    // Create new Organizations data set and parse json from the response
    var orgs Organizations
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

    if space != "" {
        fmt.Println("Space is: " + space)
    }
    if role != "" {
        fmt.Println("Role is: " + role)
    }
}

func sendHttpRequest(method string, url string, querystring *url.Values) *http.Response {
    // Create new http request
    req, err := http.NewRequest(method, url, nil)
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