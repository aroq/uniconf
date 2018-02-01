# build stage
FROM golang:1.9.3-alpine3.7 AS build-env
WORKDIR /go/src/github.com/aroq/uniconf
COPY . .
RUN apk update && apk upgrade && apk add --no-cache git
RUN go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o uniconf.docker.build .

# final stage
FROM jenkins/jenkins:2.73.3-alpine
COPY --from=build-env /go/src/github.com/aroq/uniconf/uniconf.docker.build /uniconf/uniconf
