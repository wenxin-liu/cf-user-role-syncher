package main

import (
    "os"
    "fmt"
    "log"
    "encoding/json"
    "io/ioutil"

    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
    "google.golang.org/api/admin/directory/v1"
)


func configFromFile(config *Config, file string) {
    // Read configuration
    confcontent, err := ioutil.ReadFile(file)
    if err != nil { log.Fatal(err) }
    // Parse json into Config object
    err = json.Unmarshal(confcontent, &config)
    if err != nil { log.Fatal(err) }
}


// Returns oauth.Config (e.g. Google oauth endpoint)
func getOauthConfig(file string) *oauth2.Config {
    // Read the local oauth credentials file
    b, err := ioutil.ReadFile(file)
    if err != nil {
        log.Fatalf("Unable to read client secret file: %v", err)
    }
    // If modifying these scopes, delete your previously saved token.json.
    config, err := google.ConfigFromJSON(b, admin.AdminDirectoryGroupScope)
    if err != nil {
            log.Fatalf("Unable to parse client secret file to config: %v", err)
    }
    return config
}


// Loads existing oauth token from local file (access_key and resfresh_key)
func tokenFromFile(file string) (*oauth2.Token, error) {
    f, err := os.Open(file)
    defer f.Close()
    if err != nil {
            return nil, err
    }
    tok := &oauth2.Token{}
    err = json.NewDecoder(f).Decode(tok)
    return tok, err
}


// Starts process of getting oauth token, by authenticating on Google using a browser
func getTokenFromWeb(config *oauth2.Config, file string) error {
    // Generate URL where user needs to authenticate using his browser
    authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
    fmt.Printf("Go to the following link in your browser, then type the "+
            "authorization code here on the command line and hit enter: \n%v\n", authURL)

    // Ask the user to paste the received code on the command line
    var authCode string
    if _, err := fmt.Scan(&authCode); err != nil {
        log.Fatalf("Unable to read the pasted authorization code: %v", err)
    }

    // Exchange the authCode for an oauth token
    token, err := config.Exchange(oauth2.NoContext, authCode)
    if err != nil {
        log.Fatalf("Unable to retrieve oauth token from web: %v", err)
    }

    // Save oauth token to local file
    f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
    defer f.Close()
    if err != nil {
        log.Fatalf("Unable to write token to disk in file " + file + ": %v", err)
    }
    json.NewEncoder(f).Encode(token)
    return err
}


func getAccessTokenCF() string {
    // Read configuration
    confcontent, err := ioutil.ReadFile("/Users/glo3937/.cf/config.json")
    if err != nil { log.Fatal(err) }

    // Parse json into Config object
    var config Config
    err = json.Unmarshal(confcontent, &config)
    if err != nil { log.Fatal(err) }

    return config.AccessToken
}