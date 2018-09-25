# CF-user-role-syncher
CF-user-role-syncher automates the assignment of roles to users in CloudFoundry, by synching between a source state and CF. 

- [Why is this app necessary?](#why-is-this-app-necessary)
- [Installation and configuration](#installation-and-configuration)
- [How to run locally?](#how-to-run-locally)
- [How the app works in detail](#how-the-app-works-in-detail)
- [Specifics for running in halfpipe (Springer Nature only)](#specifics-for-running-in-halfpipe-springer-nature-only)

## Why is this app necessary?
In the current iteration of CloudFoundry single sign on, UAA handles only authentication, but not authorization. Effectively, this means that while you can log into CloudFoundry via SSO, you cannot perform any actions once logged in, because you are not assigned any roles and rights automatically. This app aims to automate this missing functionality. In other words, cf-user-role-syncher automates the assignment of roles for users in CloudFoundry.

Currently, the source state used by this app to sync to CloudFoundry is Google Groups.

## Installation and configuration
#### 1. Create source state in Google Groups
In Google groups, for org level roles, group name must follow the format below:

  > *groupprefix__CForgname__rolename@yourdomain.com* for org roles.  
  > Possible org role names are: `orgmanager`, `billingmanager`, `auditor`  
  > e.g. cfroles__engineering-enablement__orgmanager@springernature.com. 
  
Then add users who belong to CF org Engineering Enablement with user role orgmanager to this group.
  
For space level roles, group name must follow the format below: 

  > *groupprefix__CForgname__spacename__rolename@yourdomain.com* for space roles.  
  > Possible space role names are: `spacemanager`, `spacedeveloper`, `spaceauditor`  
  > e.g. cfroles__engineering-enablement__live__spacedeveloper@springernature.com

Then add users who belong to CF org Engineering Enablement and space Live with user role spacedeveloper to this group.

#### 2. Build the app
- Clone the repo
- `cd cf-user-role-syncher/gmapper`
- `go build gmapper.go`

> This app is using the module feature from Go 1.11. Therefore, Go 1.11 or up is required to build. If you are building from inside your $GOPATH, please keep [these](https://github.com/golang/go/wiki/Modules#installing-and-activating-module-support) instructions in mind.

This will build the binary (filename: *gmapper*) in the current directory.

#### 3. Configure the environment variables
CF-user-role-syncher needs a couple of environment variables to be set. In short it needs to know:
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
| GOOGLEREDIRECTURI | urn:ietf:wg:oauth:2.0:oob | This is the first redirect URI provided by Google when you download your Oauth client ID and Secret from Google. Probably a fixed value until Google decides to change this. |
| GOOGLEAUTHURI | https://accounts.google.com/o/oauth2/auth | Fixed value. This will only change when Google decides to change its Oauth endpoints. |
| GOOGLETOKENURI | https://www.googleapis.com/oauth2/v3/token | Fixed value. This will only change when Google decides to change its Oauth endpoints. |
| GOOGLEOAUTHSCOPE | https://www.googleapis.com/auth/admin.directory.group | Fixed value. This will only change when Google decides to change its Oauth scope names. |
| GOOGLEACCESSTOKEN | dg26.s2iuwxguiw-wiwcvcxh | [How to get this?](OAUTH.md#oauth-refresh-token-for-google) |
| GOOGLEREFRESHTOKEN | hwqec/wqdc82dwqu21d12jw-21 | [How to get this?](OAUTH.md#oauth-refresh-token-for-google) |
| GOOGLETOKENTYPE | Bearer | [How to get this?](OAUTH.md#oauth-refresh-token-for-google) |

## How to run locally?
There is a *source* file `set-env-vars` provided in the repository which sets all the required environment variables. This will fetch its values from:
- Your local cf config file (`~/.cf/config.json`). Make sure you are logged in to CF as Admin.
- Downloaded Google Client credentials file (`credentials.json`). Get one [here](OAUTH.md#oauth-client-credentials-for-google).
- Generated Google Oauth Token file (`token.json`) Get one [here](OAUTH.md#oauth-refresh-token-for-google). 

By default, your CF configuration will be saved in `~/.cf/.` Please save `credentials.json` and `token.json` in the same directory as the `set-env-vars` file.

Then run:
```bash
source /path/to/gmapper/repo/set-env-vars
```
With the environment variables set you can now run the *cf-user-role-syncher* binary.

## How the app works in detail
The app performs the steps below:
- Search in your GSuite Directory for groups starting with the defined group name prefix. The prefix is meant to identify the groups that are used for CF authorization. For example, search for all groups starting with *cfrole__*. This allows for more groups to exist in Google Groups, not all used for managing CF authorization.
- Iterate over every found group. For every group do:
  - Is it about an org role or a space role? The information is extracted from the structure of the group name, e.g. groupprefix__CForgname__rolename@yourdomain.com or groupprefix__CForgname__spacename__rolename@yourdomain.com
  - Fetch the members from the group. 
  - Even with sso, uaa requires an actual user account to be present. Therefore, cf-user-role-syncher checks if a group member already exists as user in uaa, using the email address as username. If not, the user will be created.
  - The org or space role is assigned to the user.

## Specifics for running in halfpipe (Springer Nature only)
Halfpipe is the CI system within Springer Nature. The pipeline definition is configured in `gmapper/.halfpipe.io`. Currently the pipeline is configured to run on a scheduled basis every 15 minutes. This makes sure Google Group members are continuously mapped to their respective roles in CF. However, since the app runs are scheduled, do keep in mind there's a delay for the role mapping between Google and CF of up to 15 minutes.

The needed environment variables are set in Vault. The Vault path is: `springernature/engineering-enablement/cf-user-role-syncher`
