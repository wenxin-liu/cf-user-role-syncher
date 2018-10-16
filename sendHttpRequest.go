package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/SpringerPE/cf-user-role-syncher/gmapper/token"
)

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
	// Set Headers
	// uaaresponse := token.GetTokenFromUaa()
	// req.Header.Add("Authorization", token.UnmarshalJson(uaaresponse))
	req.Header.Add("Authorization", cfAccessToken)
	if (method == "POST") || (method == "PUT") {
		req.Header.Add("Content-Type", "application/json")
	}
	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error while executing HTTP request: %v\n", err)
	}
	// The Oauth AccessToken could be expired. If so, we do one try to get a new one and retry the request
	if resp.StatusCode == 401 {
		log.Println("Received HTTP 401 response.")
		type ErrorCode struct {
			ErrorCode string `json:"error_code"`
		}
		var errorCode ErrorCode
		// Try to read the CF error_code from the response body
		if err := json.NewDecoder(resp.Body).Decode(&errorCode); err != nil {
			log.Printf("Error while reading error_code: %v\n", err)
		} else {
			if errorCode.ErrorCode == "CF-InvalidAuthToken" {
				log.Println("CF OAuth Access Token has expired. Will try to get new Access Token.")
				// Get new AccessToken
				cfAccessToken, err = token.GetCfAccessToken()
				if err != nil {
					log.Fatalf("Failed getting a new CF Access Token: %v", err) // Exit app
				}
				// Reset the Authorization header
				req.Header.Set("Authorization", cfAccessToken)
				// Retry the original request
				resp, err = client.Do(req)
				if err != nil {
					log.Fatalf("Error while retrying HTTP request with new CF Access Token: %v\n", err) // Exit app
				} else if resp.StatusCode == 401 {
					log.Fatalln("Retrying the original HTTP request with new Access Token still results in HTTP 401.") // Exit app
				}
			} // End if (errorCode check)
		} // End if (Successful response body parsing)
	} // End if (StatusCode = 401)
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
