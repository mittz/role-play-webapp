# Overview

This provides resources for the competition of `Role Play Series - Web Application Engineer`.

# Prerequisites

* Golang: 1.17 or later
* Docker Engine: 20.10.13 or later
* Docker Compose: 1.29.2 or later
* Terraform:

# Deployment for Web Application on Google Cloud

```shell
$ git clone git@github.com:mittz/role-play-webapp.git
$ cd provisioning
$ terraform init
$ terraform apply
```

For more details on the web application deployment, see [README.md](/webapp/README.md).

# Contribution

Please refer to [CONTRIBUTING.md](/CONTRIBUTING.md) for details.

# Assets

- [webapp](/webapp/): Resources to deploy the web application
- [benchmark](/benchmark/): Resources to deploy the benchmark application
- [provisioning](/provisioning/): Resources to provision the web and benchmark applications
