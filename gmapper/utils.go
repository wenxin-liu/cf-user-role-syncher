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


// func getOauthConfig() *oauth2.Config {

// }


// func getOauthToken() *oauth2.Token {

// }


func getConfigFromFile(config *Config, file string) {
    // Read configuration
    confcontent, err := ioutil.ReadFile(file)
    if err != nil { log.Fatal(err) }
    // Parse json into Config object
    err = json.Unmarshal(confcontent, &config)
    if err != nil { log.Fatal(err) }
}


// Returns oauth.Config (e.g. Google oauth endpoint)
func getOauthConfigFromFile(file string) *oauth2.Config {
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
func getOauthTokenFromFile(file string) (*oauth2.Token, error) {
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


func contains(arr []string, str string) bool {
   for _, a := range arr {
      if a == str {
         return true
      }
   }
   return false
}

//
//func BytesToString1(data []byte) string {
//    return string(data[:])
//}

//func getAccessTokenCF() string {
//    body := strings.NewReader(`client_id=cf&client_secret=&grant_type=refresh_token&refresh_token=eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleS0xIiwidHlwIjoiSldUIn0.eyJqdGkiOiI3Njg0NGZlMzVlMmY0NTE3YjFkNjkyNTkyNmQ0NGViMi1yIiwic3ViIjoiNzgwY2Q0NmQtMmZiMS00N2JkLWJmZjctYzRlYjYyMDgzMjU5Iiwic2NvcGUiOlsib3BlbmlkIiwicm91dGluZy5yb3V0ZXJfZ3JvdXBzLndyaXRlIiwic2NpbS5yZWFkIiwiY2xvdWRfY29udHJvbGxlci5hZG1pbiIsInVhYS51c2VyIiwicm91dGluZy5yb3V0ZXJfZ3JvdXBzLnJlYWQiLCJjbG91ZF9jb250cm9sbGVyLnJlYWQiLCJwYXNzd29yZC53cml0ZSIsImNsb3VkX2NvbnRyb2xsZXIud3JpdGUiLCJuZXR3b3JrLmFkbWluIiwiZG9wcGxlci5maXJlaG9zZSIsInNjaW0ud3JpdGUiXSwiaWF0IjoxNTM2MTQwNjI5LCJleHAiOjE1Mzg3MzI2MjksImNpZCI6ImNmIiwiY2xpZW50X2lkIjoiY2YiLCJpc3MiOiJodHRwczovL3VhYS5zbnBhYXMuZXUvb2F1dGgvdG9rZW4iLCJ6aWQiOiJ1YWEiLCJncmFudF90eXBlIjoicGFzc3dvcmQiLCJ1c2VyX25hbWUiOiJhZG1pbiIsIm9yaWdpbiI6InVhYSIsInVzZXJfaWQiOiI3ODBjZDQ2ZC0yZmIxLTQ3YmQtYmZmNy1jNGViNjIwODMyNTkiLCJyZXZfc2lnIjoiZDE5OWNjNzQiLCJhdWQiOlsic2NpbSIsImNsb3VkX2NvbnRyb2xsZXIiLCJwYXNzd29yZCIsImNmIiwidWFhIiwib3BlbmlkIiwiZG9wcGxlciIsInJvdXRpbmcucm91dGVyX2dyb3VwcyIsIm5ldHdvcmsiXX0.DbQHbYLzlNa-CohEBAcb6NtzoZ05oelgkbkOs09vPq5tZJ-TvxHU4evmmKObF9i9-3pmskWAN5RXX6l44MJhN0AdbitLD5O-lFDJes04ygryA1sR7-Ux843agEl89QdFxfw0fDuIblDj3HYYuUApwK3ihIxMfKzXbBbwm2lfOADdH4Rwg_Gsi5_lX-3axX4K5QUnhJS-MC8c40OX6lQE395aAPhW4XeFW_neIqHrMddzkSsvSzvDtVcY16MAfsgNiziSIb4LQLELPSltF6tJm4PJdz5gfrzpKcOLfUWKTfJiXrL5O-3V8xCU0iJsNwCcERlPbu-8SyE20IBieOTj4A`)
//    req, err := http.NewRequest("POST", "https://login.snpaas.eu/oauth/token", body)
//    if err != nil {
//        // handle err
//    }
//    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
//    req.Header.Set("Accept", "application/json")
//
//    resp, err := http.DefaultClient.Do(req)
//    if err != nil {
//        // handle err
//    }
//    defer resp.Body.Close()
//
//    body1, err := ioutil.ReadAll(resp.Body)
//
//    body2 := BytesToString1(body1)
//    fmt.Println(body2)
//
//    return body2
//
//}