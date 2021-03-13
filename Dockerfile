#
# build container
#
FROM golang:1.16-alpine as builder
WORKDIR /go/src/github.com/oliver006/rethinkdb_exporter/

ADD .  /go/src/github.com/oliver006/rethinkdb_exporter/

ARG GOARCH="amd64"
ARG SHA1="[no-sha]"
ARG TAG="[no-tag]"

RUN apk --no-cache add ca-certificates
RUN BUILD_DATE=$(date +%F-%T) && CGO_ENABLED=0 GOOS=linux GOARCH=$GOARCH go build -o /bin/rethinkdb_exporter \
    -ldflags  "-s -w -extldflags \"-static\" -X main.BuildVersion=$TAG -X main.BuildCommitSha=$SHA1 -X main.BuildDate=$BUILD_DATE" .

#
# scratch release container
#
FROM scratch as scratch

COPY --from=builder /bin/rethinkdb_exporter /bin/rethinkdb_exporter
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

# Run as non-root user for secure environments
USER 59000:59000

EXPOSE     9123
ENTRYPOINT [ "/bin/rethinkdb_exporter" ]
