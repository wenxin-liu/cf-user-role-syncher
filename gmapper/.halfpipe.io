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
    image: golang
