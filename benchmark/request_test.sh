#!/bin/bash

curl -X POST http://localhost:8081/benchmark -d "endpoint=http://192.168.1.1:8080&userkey=675E9C3A-6E0B-4144-81A1-F0FFBC203DC8&project_id=user001"
curl -X POST http://localhost:8081/benchmark -d "endpoint=http://192.168.1.2:8080&userkey=0BF092B6-2EAB-4888-A6EE-5AB8DB3D2281&project_id=user002"
curl -X POST http://localhost:8081/benchmark -d "endpoint=http://192.168.1.3:8080&userkey=94DAC332-BE4C-47FE-98BF-741BE39000DE&project_id=user003"
curl -X POST http://localhost:8081/benchmark -d "endpoint=http://192.168.1.4:8080&userkey=7B753231-BF70-456C-A37A-57F33F900A3C&project_id=user004"
curl -X POST http://localhost:8081/benchmark -d "endpoint=http://192.168.1.5:8080&userkey=88469504-4CC6-41A1-8703-C06DD25A0B03&project_id=user005"
curl -X POST http://localhost:8081/benchmark -d "endpoint=http://192.168.1.6:8080&userkey=EE0AA9F3-A62C-4E1A-90E7-348A56C179DE&project_id=user006"