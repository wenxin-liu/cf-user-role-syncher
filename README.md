# Gmapper
The aim of Gmapper is to map members (users) of a Google **Group** to its respective **Role** in CloudFoundry. This automates the assignment of Org and Space roles to individual users.

- [Why was Gmapper created?](#why-was-gmapper-created)
- [How to build?](#how-to-build)
- [What do you need to feed into Gmapper?](#what-do-you-need-to-feed-into-gmapper)
- [What does the app really do?](#what-does-the-app-really-do)
- [How to run locally?](#how-to-run-locally)
- [Specifics for running in halfpipe](#specifics-for-running-in-halfpipe)


## Why was Gmapper created?
When single sign on (SSO) is implemented on CloudFoundry (CF), this means there is an authentication provider configured for UAA. Within Springer Nature we decided to use Google as provider using OpenID Connect as the SSO protocol. UAA handles the authentication (who are you?), but it does not handle the authorization (what are you allowed to do?). Every user needs to get an Org and/or Space role assigned within CF before it can do anything. Therefore, Gmapper was created to act as a complementary component to CF SSO in order to automate authorization.

## How to build?
- Clone the repo
- `cd cf-google-sso-authorization-handler/gmapper`
- `go build gmapper.go`

> This app is using the module feature from Go 1.11. Therefore, Go 1.11 or up is required to build. If you are building from inside your $GOPATH, please keep [these](https://github.com/golang/go/wiki/Modules#installing-and-activating-module-support) instructions in mind.

## What do you need to feed into Gmapper?
Gmapper needs a couple of environment variables to be set. In short it needs to know:
- The environment (e.g. endpoints from Google and CF)
- Oauth credentials for both Google and CF

Environment variables overview:

| Variable Name | Example Value | Notes |
| ------------- | ------------- | ----- |
| CFAPIENDPOINT | https://api.mycfdomain.org |
| UAAENDPOINT | https://uaa.mycfdomain.org |
| UAASSOPROVIDER | google | This is how you named the configured OpenID Connect provider in uaa |
| OAUTHCFREFRESHTOKEN | eyJhbGciOiJSUzI1NiIs | [How to get this?](OAUTH.md#oauth-refresh-token-for-cf) |
| GOOGLECLIENTID | 873e7823-ajhgsy652w.apps.googleusercontent.com | [How to get this?](OAUTH.md#oauth-client-credentials-for-google) |
| GOOGLECLIENTSECRET | qwhk3f9ewy823fuw | [How to get this?](OAUTH.md#oauth-client-credentials-for-google) |
| GOOGLEREDIRECTURI | urn:ietf:wg:oauth:2.0:oob | This is the first redirect URI provided by Google when you download your Oauth client ID and Secret from Google |
| GOOGLEAUTHURI | https://accounts.google.com/o/oauth2/auth | Fixed value. This will only change when Google decides to change its Oauth endpoints. |
| GOOGLETOKENURI | https://www.googleapis.com/oauth2/v3/token | Fixed value. This will only change when Google decides to change its Oauth endpoints. |
| GOOGLEOAUTHSCOPE | https://www.googleapis.com/auth/admin.directory.group | Fixed value. This will only change when Google decides to change its Oauth scope names. |
| GOOGLEACCESSTOKEN | dg26.s2iuwxguiw-wiwcvcxh | [How to get this?](OAUTH.md#oauth-refresh-token-for-google) |
| GOOGLEREFRESHTOKEN | hwqec/wqdc82dwqu21d12jw-21 | [How to get this?](OAUTH.md#oauth-refresh-token-for-google) |
| GOOGLETOKENTYPE | Bearer | [How to get this?](OAUTH.md#oauth-refresh-token-for-google) |

## What does the app really do?


## How to run locally?
There is a *source* file provided in the repository which sets all the required environment variables. This will fetch its values from:
- Your local cf config file (`~/.cf/config.json`). Make sure you are logged in to CF as Admin.
- Downloaded Google Client credentials file (`./credentials.json`)
- Generated Google Oauth Tokens file (`./token.json`)
First follow [these](OAUTH.md) instructions to get the `credentials.json` and `token.json` files. From within the directory where you saved these two files run:
```bash
source /path/to/gmapper/repo/set-env-vars
```
With the environment variables set you can now run the *gmapper* binary.


## Specifics for running in halfpipe
Halfpipe is the CI system within Springer Nature. The pipeline definition is configured in `gmapper/.halfpipe.io`. Currently the pipeline is configured to run on a scheduled basis. This makes sure Google Group members are continuously mapped to their respective roles in CF. However, since the app runs are scheduled, do keep in mind the role mapping between Google and CF is **not** instant.

The needed environment variables are set in Vault. The Vault path is: `springernature/engineering-enablement/gsuite-cf-roles-mapper`
