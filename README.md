# Assets

- app/: Resources for application layer
- database/: Resources for database layer
- provisioning/: Resources to provision this web application
- database.json: Configuration file to setup the database
- initdata.json: Data to initiatize the database
- main.go: Main file to run the web application
- main_test.go: Test for main.go

# Development

For development, you can build your own test environment by the following steps.

```
$ git clone https://github.com/mittz/role-play-webapp
```

## Web App

## Run PostgreSQL Container

```shell
$ cd role-play-webapp/provisioning
$ docker-compose up -d
```

## (Optional) Update Database Connection Config

If you would like to change the settings like database host and port, update the `role-play-webapp/database.json` file.

## Run Web App

```shell
$ cd role-play-webapp
$ go run main.go
```

## (Optional) Init Database

If you would like to reset the data in the PostgreSQL database, run the following command. This endpoint `/admin/init` is also used by the scoring server.

```shell
$ curl -X POST http://localhost:8080/admin/init
```

## Access Web App

Access to `http://localhost:8080/` and see if you can see the web site.

# Contribution