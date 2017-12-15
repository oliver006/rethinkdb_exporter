FROM alpine:latest
MAINTAINER Oliver <o@21zoo.com>

ENV GOPATH /go
COPY . /go/src/github.com/oliver006/rethinkdb_exporter

RUN apk add --update -t build-deps go git mercurial make \
    && apk add -u musl musl-dev && rm -rf /var/cache/apk/* \
    && cd /go/src/github.com/oliver006/rethinkdb_exporter \
    && go get && go build && cp rethinkdb_exporter /bin/rethinkdb_exporter \
    && rm -rf /go && apk del --purge build-deps

EXPOSE     9123
ENTRYPOINT [ "/bin/rethinkdb_exporter" ]
