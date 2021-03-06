# How to get the required Oauth Tokens, Client ID and Secret
This document provides instructions on how to get all the Oauth related configuration for the Gmapper app to run.

## Create credentials for CF
Gmapper authenticates to the CF api with Oauth. This requires to send along an *Authorization* header with every api call containing a valid Oauth Access Token. An Access Token will expire after a while. The app fetches a new Access Token using a CF username and password. This user does need admin permissions. This is how you create such a user:

```bash
#First create a user
cf create-user automation.user@mydomain.com SecretP@assw0rd

# Use the uaa cli to assign admin scopes
uaac member add cloud_controller.admin automation.user@mydomain.com
uaac member add uaa.admin automation.user@mydomain.com
uaac member add scim.read automation.user@mydomain.com
uaac member add scim.write automation.user@mydomain.com
```

Both the username and password need to be set using environment variables for the app to work.

### Using the username/password for running the app locally
This project contains a simple [script](README.md#how-to-run-locally) to set all the needed environment variables locally. If you like to use this script, please save your CF credentials in a new json file called `cfcredentials.json` in the root of the project.

```json
{
 "username": "automation.user@mydomain.com",
 "password": "SecretP@assw0rd"   
}
```


## Oauth Client Credentials for Google
To be able to call the [Google Directory API](https://developers.google.com/admin-sdk/directory/) for searching Google Groups we need Oauth credentials. In the world of Oauth *gmapper* is the client application which needs to be a *known* app to Google. The way this works in Oauth is by obtaining a Client ID and Secret from Google. Doing this is fairly simple:
- Register the app using the [Google API Console](https://console.developers.google.com/).
  - Click *credentials* in the left hand menu
  - Click *Create Credentials*. Choose for *Oauth Client ID*.
  - Application type is *Other*
- Your new key will show under the *OAuth 2.0 client IDs* paragraph
- Download the credentials. Save the file as `credentials.json`

The *credentials.json* file contains information for the gmapper app such as Googles Oauth endpoints, the client ID and the client secret. If you need to set the environment variables manually for gmapper (e.g. in Vault), these are the things you need:
```bash
# Client ID
cat credentials.json | jq -r .installed.client_id
# Client Secret
cat credentials.json | jq -r .installed.client_secret
# Auth URI
cat credentials.json | jq -r .installed.auth_uri
# Token URI
cat credentials.json | jq -r .installed.token_uri
# Redirect URI
cat credentials.json | jq -r .installed.redirect_uris[0]
```


## Oauth Refresh Token for Google
In the above [step](#oauth-client-credentials-for-google) we only registered an app at Google. This does not provide us yet with an Oauth Token to authorize Google API calls. This section explains how to get the necessary Token data for gmapper.

First make sure the `credentials.json` file from the previous [step](#oauth-client-credentials-for-google) is in the same directory as the `gmapper` binary. Now run the following command:
```bash
gmapper token
```
- This will start the process of obtaining a valid Oauth Token. The gmapper app will ask to open an URL in you browser.
- Copy the URL and open in a browser. The webpage is from Google asking you to sign in with your Google account. Sign in!
> Use a newly created dedicated user which is a GSuite admin in your GSuite Directory. Being a GSuite Admin is mandatory. Group read-only permission is sufficient.
- Google displays a consent screen, asking you to authorize the application to request *group* data on you behalf. Approve this request.
- After you approved, Google will return a code. Copy and paste the code back on the cli.
- Hit Enter
- A file `token.json` should now be created.

If you need to set the environment variables manually for gmapper (e.g. in Vault), these are the things you need:
```bash
# Access Token
cat token.json | jq -r .access_token
# Refresh Token
cat token.json | jq -r .refresh_token
# Token Type
cat token.json | jq -r .token_type
```
