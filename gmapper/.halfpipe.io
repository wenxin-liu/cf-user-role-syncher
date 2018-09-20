team: engineering-enablement
pipeline: gsuite-cf-roles-mapper-gmapper
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
    CFAPIENDPOINT: ((gsuite-cf-roles-mapper.CFAPIENDPOINT))
    UAAENDPOINT: ((gsuite-cf-roles-mapper.UAAENDPOINT))
    UAASSOPROVIDER: ((gsuite-cf-roles-mapper.UAASSOPROVIDER))
    OAUTHCFREFRESHTOKEN: ((gsuite-cf-roles-mapper.OAUTHCFREFRESHTOKEN))
    GOOGLEREDIRECTURI: ((gsuite-cf-roles-mapper.GOOGLEREDIRECTURI))
    GOOGLEAUTHURI: ((gsuite-cf-roles-mapper.GOOGLEAUTHURI))
    GOOGLETOKENURI: ((gsuite-cf-roles-mapper.GOOGLETOKENURI))
    GOOGLECLIENTID: ((gsuite-cf-roles-mapper.GOOGLECLIENTID))
    GOOGLECLIENTSECRET: ((gsuite-cf-roles-mapper.GOOGLECLIENTSECRET))
    GOOGLEOAUTHSCOPE: ((gsuite-cf-roles-mapper.GOOGLEOAUTHSCOPE))
    GOOGLEACCESSTOKEN: ((gsuite-cf-roles-mapper.GOOGLEACCESSTOKEN))
    GOOGLEREFRESHTOKEN: ((gsuite-cf-roles-mapper.GOOGLEREFRESHTOKEN))
    GOOGLETOKENTYPE: ((gsuite-cf-roles-mapper.GOOGLETOKENTYPE))

