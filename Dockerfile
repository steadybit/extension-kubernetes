# syntax=docker/dockerfile:1

##
## Build
##
FROM --platform=$BUILDPLATFORM golang:1.20-alpine AS build

ARG TARGETOS TARGETARCH
ARG NAME
ARG VERSION
ARG REVISION
ARG ADDITIONAL_BUILD_PARAMS
ARG SKIP_LICENSES_REPORT

WORKDIR /app

RUN apk add build-base
COPY go.mod ./
COPY go.sum ./
RUN go mod download

RUN wget -P /usr/bin "https://storage.googleapis.com/kubernetes-release/release/$(wget -O - https://dl.k8s.io/release/stable.txt)/bin/linux/${TARGETARCH}/kubectl" && chmod a+x /usr/bin/kubectl

COPY . .

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="\
    -X 'github.com/steadybit/extension-kit/extbuild.ExtensionName=${NAME}' \
    -X 'github.com/steadybit/extension-kit/extbuild.Version=${VERSION}' \
    -X 'github.com/steadybit/extension-kit/extbuild.Revision=${REVISION}'" \
    -o ./extension \
    ${ADDITIONAL_BUILD_PARAMS}
RUN if [[ -z "$SKIP_LICENSES_REPORT" ]] ; then make licenses-report ; else mkdir /app/licenses ; fi

##
## Runtime
##
FROM alpine:3.18

LABEL "steadybit.com.discovery-disabled"="true"

ARG USERNAME=steadybit
ARG USER_UID=10000

RUN adduser -u $USER_UID -D $USERNAME

USER $USERNAME

WORKDIR /

COPY --from=build app/extension /extension
COPY --from=build /usr/bin/kubectl /usr/bin/kubectl
COPY --from=build /app/licenses /licenses

EXPOSE 8088
EXPOSE 8089

ENTRYPOINT ["/extension"]
