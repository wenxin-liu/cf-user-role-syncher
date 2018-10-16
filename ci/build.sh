#!/bin/sh

env GOOS=linux GOARCH=amd64 go build -o gmapper-linux

echo "\n  env:" >> manifest.yml
echo "    CFAPIENDPOINT: ${CFAPIENDPOINT}" >> manifest.yml
echo "    UAAENDPOINT: ${UAAENDPOINT}" >> manifest.yml
echo "    UAASSOPROVIDER: ${UAASSOPROVIDER}" >> manifest.yml
echo "    OAUTHCFREFRESHTOKEN: ${OAUTHCFREFRESHTOKEN}" >> manifest.yml
echo "    GOOGLEREDIRECTURI: ${GOOGLEREDIRECTURI}" >> manifest.yml
echo "    GOOGLEAUTHURI: ${GOOGLEAUTHURI}" >> manifest.yml
echo "    GOOGLETOKENURI: ${GOOGLETOKENURI}" >> manifest.yml
echo "    GOOGLECLIENTID: ${GOOGLECLIENTID}" >> manifest.yml
echo "    GOOGLECLIENTSECRET: ${GOOGLECLIENTSECRET}" >> manifest.yml
echo "    GOOGLEOAUTHSCOPE: ${GOOGLEOAUTHSCOPE}" >> manifest.yml
echo "    GOOGLEACCESSTOKEN: ${GOOGLEACCESSTOKEN}" >> manifest.yml
echo "    GOOGLEREFRESHTOKEN: ${GOOGLEREFRESHTOKEN}" >> manifest.yml
echo "    GOOGLETOKENTYPE: ${GOOGLETOKENTYPE}" >> manifest.yml