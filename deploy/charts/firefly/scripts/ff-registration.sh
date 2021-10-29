#!/bin/sh

apk add curl jq

until STATUS=$(curl ${FF_URL}/api/v1/status); do
  echo "Waiting for FireFly..."
  sleep 5
done

if [ `echo $STATUS | jq -r .org.registered` != "true" ]; then

  echo "Registering organization"
  HTTP_CODE=`curl --silent --output /dev/stderr --write-out "%{http_code}" \
    -X POST -d '{}' -H 'Content-Type: application/json' \
    "${FF_URL}/api/v1/network/organizations/self?confirm"`
  if [ "$HTTP_CODE" -ne 200 ]; then
    echo "Failed to register with code ${HTTP_CODE}"
    exit 1
  fi

fi

if [ `echo $STATUS | jq -r .node.registered` != "true" ]; then

  echo "Registering node"
  HTTP_CODE=`curl --silent --output /dev/stderr --write-out "%{http_code}" \
    -X POST -d '{}' -H 'Content-Type: application/json' \
    "${FF_URL}/api/v1/network/nodes/self?confirm"`
  if [ "$HTTP_CODE" -ne 200 ]; then
    echo "Failed to register with code ${HTTP_CODE}"
    exit 1
  fi

else

  echo "Already registered. Nothing to do"

fi
