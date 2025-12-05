# syntax=docker/dockerfile:1

##
## Build
##
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG NAME
ARG VERSION
ARG REVISION
ARG ADDITIONAL_BUILD_PARAMS
ARG SKIP_LICENSES_REPORT=false
ARG VERSION=unknown
ARG REVISION=unknown

WORKDIR /app

RUN apk add build-base
COPY go.mod ./
COPY go.sum ./
RUN go mod download

RUN KUBECTL_VERSION="$(curl --proto "=https" -fsSL https://dl.k8s.io/release/stable.txt)" \
    && curl --proto "=https" -fsSL -o /usr/bin/kubectl "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${TARGETARCH}/kubectl" \
    && chmod a+x /usr/bin/kubectl

COPY . .

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="\
    -X 'github.com/steadybit/extension-kit/extbuild.ExtensionName=${NAME}' \
    -X 'github.com/steadybit/extension-kit/extbuild.Version=${VERSION}' \
    -X 'github.com/steadybit/extension-kit/extbuild.Revision=${REVISION}'" \
    -o ./extension \
    ${ADDITIONAL_BUILD_PARAMS}
RUN make licenses-report

##
## Runtime
##
FROM alpine:3.22

ARG VERSION=unknown
ARG REVISION=unknown

LABEL "steadybit.com.discovery-disabled"="true"
LABEL "version"="${VERSION}"
LABEL "revision"="${REVISION}"
RUN echo "$VERSION" > /version.txt && echo "$REVISION" > /revision.txt

ARG USERNAME=steadybit
ARG USER_UID=10000

RUN adduser -u $USER_UID -D $USERNAME

USER $USER_UID

WORKDIR /

COPY --from=build app/extension /extension
COPY --from=build /usr/bin/kubectl /usr/bin/kubectl
COPY --from=build /app/licenses /licenses

EXPOSE 8088
EXPOSE 8089

ENTRYPOINT ["/extension"]
