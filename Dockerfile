# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.18-alpine AS build

ARG TARGETARCH
ARG NAME
ARG VERSION
ARG REVISION

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN wget -P /usr/bin "https://storage.googleapis.com/kubernetes-release/release/$(wget -O - https://dl.k8s.io/release/stable.txt)/bin/linux/${TARGETARCH}/kubectl"
RUN chmod a+x /usr/bin/kubectl

COPY . .

RUN go build \
    -ldflags="\
    -X 'github.com/steadybit/extension-kit/extbuild.ExtensionName=${NAME}' \
    -X 'github.com/steadybit/extension-kit/extbuild.Version=${VERSION}' \
    -X 'github.com/steadybit/extension-kit/extbuild.Revision=${REVISION}'" \
    -o /extension-kubernetes

##
## Runtime
##
FROM alpine:3.16

ARG USERNAME=steadybit
ARG USER_UID=1000

RUN adduser -u $USER_UID -D $USERNAME

USER $USERNAME

WORKDIR /

COPY --from=build /extension-kubernetes /extension-kubernetes
COPY --from=build /usr/bin/kubectl /usr/bin/kubectl

EXPOSE 8088

ENTRYPOINT ["/extension-kubernetes"]
