NAME=scstore
VERSION=1.0.0

all:
	$(MAKE) tests
	$(MAKE) build
	$(MAKE) start

build:
	$(MAKE) build-app
	$(MAKE) build-container

build-app:
	CGO_ENABLED=0 GOOS=linux go build -o $(NAME) .

build-container:
	docker build -t $(NAME):$(VERSION) .

tests:
	go test -cover ./...

start:
	docker-compose up -d

start-app:
	docker-compose up -d scstore-app

start-db:
	docker-compose up -d scstore-database

stop:
	docker-compose down