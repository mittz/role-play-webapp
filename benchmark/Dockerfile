FROM golang:1.18.1-alpine3.15 as builder

ENV APP_NAME=score
ENV ROOT=/go/src/${APP_NAME}
WORKDIR ${ROOT}
RUN apk update && apk add git

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o $APP_NAME


FROM scratch

ENV APP_NAME=score
ENV ROOT=/go/src/${APP_NAME}
COPY . /
COPY --from=builder ${ROOT}/${APP_NAME} ${APP_NAME}

EXPOSE 8080
ENTRYPOINT ["/score"]