NAME=scstore
VERSION=1.0.0
DOCKER_COMPOSE_PATH=$(shell which docker-compose)
DOCKER_COMPOSE_VERSION=1.29.2

all:
	$(MAKE) tests
	$(MAKE) build
	$(MAKE) start

build:
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

upgrade-compose:
	if [ -e $(DOCKER_COMPOSE_PATH) ]; then sudo rm -f $(DOCKER_COMPOSE_PATH); fi
	sudo curl -L "https://github.com/docker/compose/releases/download/$(DOCKER_COMPOSE_VERSION)/docker-compose-$(shell uname -s)-$(shell uname -m)" -o ${DOCKER_COMPOSE_PATH} && sudo chmod 755 $(DOCKER_COMPOSE_PATH)
