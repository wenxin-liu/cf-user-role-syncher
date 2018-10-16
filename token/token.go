package token

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/admin/directory/v1"
)

// Declaration of environment variable key names
const EnvUaaSsoProvider string = "UAASSOPROVIDER"
const EnvUaaEndPoint string = "UAAENDPOINT"
const EnvCfUsername string = "CFUSERNAME"
const EnvCfPassword string = "CFPASSWORD"
const EnvGoogleRedirectUri string = "GOOGLEREDIRECTURI"
const EnvGoogleAuthUri string = "GOOGLEAUTHURI"
const EnvGoogleTokenUri string = "GOOGLETOKENURI"
const EnvGoogleClientId string = "GOOGLECLIENTID"
const EnvGoogleClientSecret string = "GOOGLECLIENTSECRET"
const EnvGoogleOAuthScope string = "GOOGLEOAUTHSCOPE"
const EnvGoogleAccessToken string = "GOOGLEACCESSTOKEN"
const EnvGoogleRefreshToken string = "GOOGLEREFRESHTOKEN"
const EnvGoogleTokenType string = "GOOGLETOKENTYPE"

// Structure for the API response
// when getting a new Access Token for CF using the Refresh Token
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	Jti          string `json:"jti"`
}

// Function for getting a new Oauth Access Token for CF using the Refresh Token
func GetCfAccessToken() (string, error) {
	// Set POST formdata paramters
	v := url.Values{}
	v.Add("client_id", "cf")
	v.Add("client_secret", "")
	v.Add("grant_type", "password")
	v.Add("username", os.Getenv(EnvCfUsername))
	v.Add("password", os.Getenv(EnvCfPassword))
	body := strings.NewReader(v.Encode())
	// Form new http request instance
	req, err := http.NewRequest("POST", os.Getenv(EnvUaaEndPoint)+"/oauth/token", body)
	if err != nil {
		return "", err
	}
	// Set http headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	// Do the actual http request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	// Parse the response
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var tokenresponse TokenResponse
	// Parse the raw response body into a TokenResponse data structure
	json.Unmarshal(bodyBytes, &tokenresponse)
	return "bearer " + tokenresponse.AccessToken, nil
}

func GenGoogleOauthToken() {
	fmt.Println("Will now generate 'token.json' file for Google Admin Directory API...")
	// Load oauth.Config (e.g. Google oauth endpoint, client_id, client_secret)
	oauthConf := GetOauthConfigFromFile("credentials.json")
	// Start oauth process on the web to get oauth token
	err := GetTokenFromWeb(oauthConf, "token.json")
	if err != nil {
		log.Fatalf("Unable to create Google oauth token: %v", err)
	} else {
		fmt.Println("token.json" + " created!")
	}
}

// Constructs a oauth2.Config object using the values from environment variables
func GetOauthConfig() *oauth2.Config {
	// TODO: Implement check on env vars
	return &oauth2.Config{
		ClientID:     os.Getenv(EnvGoogleClientId),
		ClientSecret: os.Getenv(EnvGoogleClientSecret),
		RedirectURL:  os.Getenv(EnvGoogleRedirectUri),
		Scopes:       []string{os.Getenv(EnvGoogleOAuthScope)},
		Endpoint: oauth2.Endpoint{
			AuthURL:  os.Getenv(EnvGoogleAuthUri),
			TokenURL: os.Getenv(EnvGoogleTokenUri),
		},
	}
}

// Constructs a oauth2.Token object using the values from environment variables
func GetOauthToken() *oauth2.Token {
	// TODO: Implement check on env vars
	// The AccessToken is only valid before the expiry date.
	// As we won't be updating the AccessToken environment variable every time,
	// all we care about is the RefreshToken.
	// Therefore, we just use a static expiry date
	t, _ := time.Parse(time.RFC822, "01 Jan 18 00:00 BST")
	// Return the Token
	return &oauth2.Token{
		AccessToken:  os.Getenv(EnvGoogleAccessToken),
		TokenType:    os.Getenv(EnvGoogleTokenType),
		RefreshToken: os.Getenv(EnvGoogleRefreshToken),
		Expiry:       t,
	}
}

// Returns oauth.Config (e.g. Google oauth endpoint)
// Used when generating a Google AccessToken and RefreshToken
// by function genGoogleOauthToken()
func GetOauthConfigFromFile(file string) *oauth2.Config {
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

// Starts process of getting oauth token, by authenticating on Google using a browser
func GetTokenFromWeb(config *oauth2.Config, file string) error {
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
		log.Fatalf("Unable to write token to disk in file "+file+": %v", err)
	}
	json.NewEncoder(f).Encode(token)
	return err
}
