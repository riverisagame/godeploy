#!/bin/bash
RESP=$(curl -s -X POST http://localhost:8081/api/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}')
echo "$RESP"
TOKEN=$(echo "$RESP" | jq -r .token)
echo "TOKEN=$TOKEN"
curl -s http://localhost:8081/api/projects -H "Authorization: Bearer $TOKEN"
