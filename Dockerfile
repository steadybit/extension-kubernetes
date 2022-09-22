# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.18-alpine AS build

ARG TARGETARCH

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN wget -P /usr/bin "https://storage.googleapis.com/kubernetes-release/release/$(wget -O - https://dl.k8s.io/release/stable.txt)/bin/linux/${TARGETARCH}/kubectl"
RUN chmod a+x /usr/bin/kubectl

COPY . .

RUN go build -o /extension-kubernetes

##
## Runtime
##
FROM alpine:3.16

WORKDIR /

COPY --from=build /extension-kubernetes /extension-kubernetes
COPY --from=build /usr/bin/kubectl /usr/bin/kubectl

EXPOSE 8088

ENTRYPOINT ["/extension-kubernetes"]
