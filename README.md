# Overview

This provides ...

# Prerequisites

* Golang: 1.17 or later
* Docker Engine: 20.10.13 or later
* Terraform:

# Deployment for Web Application on Google Cloud

```shell
$ cd provisioning
$ terraform init
$ terraform apply
```

# Optional: Local Deployment for Web Application

```shell
$ git clone https://github.com/mittz/role-play-webapp
$ cd role-play-webapp
$ make all
```

Access to `http://localhost` and see if you can see the web site.

# Optional: Update Application Codes

Once you update the application codes, you need to rebuild the application and update the container image by the following command.

```shell
$ make build # This runs "make build-app" and "make build-container".
```

# Optional: Update Database Connection Config

If you would like to change the configuration of the database connection like hostname and port, update the `role-play-webapp/database.json` file. Once you update the file, you need to update the container image by the following command.

```shell
$ make build-container
```

# Optional: Run web application and database separately

If you would like to run the web application and the database separately, run the following command.

**Web Application**

```shell
$ make start-app
```

**Database**

```shell
$ make start-db
```

# Optional: Init Database

If you would like to reset the data in the PostgreSQL database, run the following command. This endpoint `/admin/init` is also used by the scoring server.

```shell
$ curl -X POST http://localhost:8080/admin/init
```

# Contribution

Please refer to [CONTRIBUTING.md](/CONTRIBUTING.md) for details.

# Assets

- app/: Resources for application layer
- database/: Resources for database layer
- provisioning/: Resources to provision this web application
- database.json: Configuration file to setup the database
- initdata.json: Data to initiatize the database
- main.go: Main file to run the web application
- main_test.go: Test for main.go