#!/bin/bash

curl -X POST http://localhost:8081/benchmark -d "endpoint=http://127.0.0.1:8080&userkey=user001&project_id=user001"
curl -X POST http://localhost:8081/benchmark -d "endpoint=http://127.0.0.1:8080&userkey=user002&project_id=user002"
curl -X POST http://localhost:8081/benchmark -d "endpoint=http://127.0.0.1:8080&userkey=user003&project_id=user003"
curl -X POST http://localhost:8081/benchmark -d "endpoint=http://127.0.0.1:8080&userkey=user004&project_id=user004"
curl -X POST http://localhost:8081/benchmark -d "endpoint=http://127.0.0.1:8080&userkey=user005&project_id=user005"
curl -X POST http://localhost:8081/benchmark -d "endpoint=http://127.0.0.1:8080&userkey=user006&project_id=user006"