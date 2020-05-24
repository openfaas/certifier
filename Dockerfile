ARG goversion=1.14
ARG alpineversion=3.11

FROM teamserverless/license-check:0.3.9 as license-check

FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:$goversion as builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG GIT_COMMIT="000000"
ARG VERSION="dev"

ENV CGO_ENABLED=0
ENV GO111MODULE=on
ENV GOFLAGS=-mod=vendor

COPY --from=license-check /license-check /usr/bin/

WORKDIR /app
COPY go.mod .
COPY go.sum .
COPY tests ./tests
COPY version ./version
COPY vendor ./vendor

RUN license-check -path /app --verbose=false "Alex Ellis" "OpenFaaS Author(s)"
RUN gofmt -l -d $(find . -type f -name '*.go' -not -path "./vendor/*")
RUN CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go test -c -o certifier \
    -ldflags "\
    -X github.com/openfaas/certifier/version.Commit=$GIT_COMMIT \
    -X github.com/openfaas/certifier/version.Version=$VERSION" \
    ./tests

FROM --platform=${TARGETPLATFORM:-linux/amd64} alpine:$alpineversion as release
LABEL org.label-schema.license="MIT" \
    org.label-schema.vcs-url="https://github.com/openfaas/certifier" \
    org.label-schema.vcs-type="Git" \
    org.label-schema.name="openfaas/certifier" \
    org.label-schema.vendor="openfaas" \
    org.label-schema.docker.schema-version="1.0"

RUN apk --no-cache --update add ca-certificates

ARG USER=default
ENV HOME /home/$USER

# install sudo as root
RUN apk add --update sudo

# add new user
RUN adduser -D $USER
USER $USER
WORKDIR $HOME

COPY --from=builder /app/certifier /bin/certifier

ENTRYPOINT [ "/bin/certifier" ]
