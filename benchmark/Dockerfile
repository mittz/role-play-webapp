FROM golang:1.18.1-alpine3.15 as builder

LABEL service_name="score"
LABEL service_role="webapp"

ENV APP_NAME=score
ENV ROOT=/go/src/${APP_NAME}
WORKDIR ${ROOT}
RUN apk update && apk add git

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o $APP_NAME


FROM alpine:3.15.4

LABEL service_name="score"
LABEL service_role="webapp"

ENV APP_NAME=score
ENV ROOT=/go/src/${APP_NAME}
COPY . /
COPY --from=builder ${ROOT}/${APP_NAME} ${APP_NAME}
RUN apk add --no-cache ca-certificates

EXPOSE 8080
ENTRYPOINT ["/score"]