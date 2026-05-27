#!/bin/bash
TOKEN=$(curl -s -X POST -H "Content-Type: application/json" -d '{"username":"admin","password":"admin123"}' http://localhost:8080/api/login | grep -o '"token":"[^"]*' | cut -d'"' -f4)
echo "JWT Token: $TOKEN"
curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"project_id":"demo-project","env_id":"testing","commit_id":"master"}' http://localhost:8080/api/tasks
echo ""
