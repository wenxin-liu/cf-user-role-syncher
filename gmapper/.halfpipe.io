team: engineering-enablement
pipeline: cf-user-role-syncher
trigger_interval: 0.25h
repo:
  watched_paths:
  - gmapper
tasks:
- type: run
  name: run
  script: ./run
  docker:
    image: golang:1.11-stretch
  vars:
    CFAPIENDPOINT: ((cf-user-role-syncher.CFAPIENDPOINT))
    UAAENDPOINT: ((cf-user-role-syncher.UAAENDPOINT))
    UAASSOPROVIDER: ((cf-user-role-syncher.UAASSOPROVIDER))
    OAUTHCFREFRESHTOKEN: ((cf-user-role-syncher.OAUTHCFREFRESHTOKEN))
    GOOGLEREDIRECTURI: ((cf-user-role-syncher.GOOGLEREDIRECTURI))
    GOOGLEAUTHURI: ((cf-user-role-syncher.GOOGLEAUTHURI))
    GOOGLETOKENURI: ((cf-user-role-syncher.GOOGLETOKENURI))
    GOOGLECLIENTID: ((cf-user-role-syncher.GOOGLECLIENTID))
    GOOGLECLIENTSECRET: ((cf-user-role-syncher.GOOGLECLIENTSECRET))
    GOOGLEOAUTHSCOPE: ((cf-user-role-syncher.GOOGLEOAUTHSCOPE))
    GOOGLEACCESSTOKEN: ((cf-user-role-syncher.GOOGLEACCESSTOKEN))
    GOOGLEREFRESHTOKEN: ((cf-user-role-syncher.GOOGLEREFRESHTOKEN))
    GOOGLETOKENTYPE: ((cf-user-role-syncher.GOOGLETOKENTYPE))