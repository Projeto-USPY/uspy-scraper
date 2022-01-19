#!/bin/bash
echo "{
  \"emulators\": {
    \"firestore\": {
      \"port\": \"$PORT\",
      \"host\": \"0.0.0.0\"
    },
    \"ui\": {
      \"enabled\": true,
      \"host\": \"0.0.0.0\",
      \"port\": \"$PORT_UI\"
    }
  }
}" > firebase.json

firebase emulators:start --project $FIRESTORE_PROJECT_ID
