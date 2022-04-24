#!/bin/bash

curl -X POST http://localhost:8080/benchmark -d "endpoint=http://user001:8080&userkey=user001&project_id=user001"
curl -X POST http://localhost:8080/benchmark -d "endpoint=http://user002:8080&userkey=user002&project_id=user002"
curl -X POST http://localhost:8080/benchmark -d "endpoint=http://user003:8080&userkey=user003&project_id=user003"
curl -X POST http://localhost:8080/benchmark -d "endpoint=http://user004:8080&userkey=user004&project_id=user004"
curl -X POST http://localhost:8080/benchmark -d "endpoint=http://user005:8080&userkey=user005&project_id=user005"
curl -X POST http://localhost:8080/benchmark -d "endpoint=http://user006:8080&userkey=user006&project_id=user006"