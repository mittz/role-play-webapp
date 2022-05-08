# Local Deployment for Benchmark Server

```shell
$ cd role-play-webapp/benchmark
$ make all
```

Access to `http://localhost:8081` and see if you can see the web site.

If you face the following error, you need to upgrade the docker-compose.

```text
docker-compose up -d
ERROR: The Compose file './docker-compose.yaml' is invalid because:
services.scstore-app.depends_on contains an invalid type, it should be an array
```

You can upgrade the docker-compose by the following command:

```shell
$ make upgrade-compose
```

# Update Application Codes

Once you update the application codes, you need to rebuild the application and update the container image by the following command.

```shell
$ make build
```

# Run web application and database separately

If you would like to run the web application and the database separately, run the following command.

**Web Application**

```shell
$ make start-app
```

**Database**

```shell
$ make start-db
```

# Stop web application and database services

```shell
$ make stop
```

# Initialize Database

Delete the persistent volume declared in [docker-compose.yaml](docker-compose.yaml).