#!/usr/bin/env bash

export CGO_ENABLED=0

docker version

gox --osarch="linux/386"   -ldflags "$GO_LDFLAGS" -output "dist/rethinkdb_exporter"

echo "Build Docker images"
docker build --rm=false -t "21zoo/rethinkdb_exporter:$CIRCLE_TAG" .
docker build --rm=false -t "21zoo/rethinkdb_exporter:latest" .

docker login -e $DOCKER_EMAIL -u $DOCKER_USER -p $DOCKER_PASS
docker push "21zoo/rethinkdb_exporter:latest"
docker push "21zoo/rethinkdb_exporter:$CIRCLE_TAG"

docker build --rm=false -t "oliver006/rethinkdb_exporter:$CIRCLE_TAG" .
docker build --rm=false -t "oliver006/rethinkdb_exporter:latest" .
docker push "oliver006/rethinkdb_exporter:latest"
docker push "oliver006/rethinkdb_exporter:$CIRCLE_TAG"

echo "Done"
