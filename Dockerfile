# This Dockerfile requires DOCKER_BUILDKIT=1 to be build.
# We do not use syntax header so that we do not have to wait
# for the Dockerfile frontend image to be pulled.
FROM golang:1.21-alpine3.18 AS build

RUN apk --update add make bash git gcc musl-dev tzdata && \
  adduser -D -H -g "" -s /sbin/nologin -u 1000 user
COPY . /go/src/docker-volume-mkfs
WORKDIR /go/src/docker-volume-mkfs
RUN \
  make build-static && \
  mv docker-volume-mkfs /go/bin/docker-volume-mkfs

FROM alpine:3.18
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /go/bin/docker-volume-mkfs /
RUN apk --update --no-cache add xfsprogs
ENTRYPOINT ["/docker-volume-mkfs"]
