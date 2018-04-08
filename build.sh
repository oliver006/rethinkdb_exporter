#!/usr/bin/env bash

export CGO_ENABLED=0

echo "Building binaries"
echo ""
echo $GO_LDFLAGS

gox -rebuild --osarch="darwin/amd64"  -ldflags "$GO_LDFLAGS" -output "dist/rethinkdb_exporter" && cd dist && tar -cvzf rethinkdb_exporter-$CIRCLE_TAG.darwin-amd64.tar.gz rethinkdb_exporter && rm rethinkdb_exporter && cd ..
gox -rebuild --osarch="darwin/386"    -ldflags "$GO_LDFLAGS" -output "dist/rethinkdb_exporter" && cd dist && tar -cvzf rethinkdb_exporter-$CIRCLE_TAG.darwin-386.tar.gz   rethinkdb_exporter && rm rethinkdb_exporter && cd ..
gox -rebuild --osarch="linux/amd64"   -ldflags "$GO_LDFLAGS" -output "dist/rethinkdb_exporter" && cd dist && tar -cvzf rethinkdb_exporter-$CIRCLE_TAG.linux-amd64.tar.gz  rethinkdb_exporter && rm rethinkdb_exporter && cd ..
gox -rebuild --osarch="linux/386"     -ldflags "$GO_LDFLAGS" -output "dist/rethinkdb_exporter" && cd dist && tar -cvzf rethinkdb_exporter-$CIRCLE_TAG.linux-386.tar.gz    rethinkdb_exporter && rm rethinkdb_exporter && cd ..
gox -rebuild --osarch="netbsd/amd64"  -ldflags "$GO_LDFLAGS" -output "dist/rethinkdb_exporter" && cd dist && tar -cvzf rethinkdb_exporter-$CIRCLE_TAG.netbsd-amd64.tar.gz rethinkdb_exporter && rm rethinkdb_exporter && cd ..
gox -rebuild --osarch="netbsd/386"    -ldflags "$GO_LDFLAGS" -output "dist/rethinkdb_exporter" && cd dist && tar -cvzf rethinkdb_exporter-$CIRCLE_TAG.netbsd-386.tar.gz   rethinkdb_exporter && rm rethinkdb_exporter && cd ..
gox -rebuild --osarch="windows/amd64" -ldflags "$GO_LDFLAGS" -output "dist/rethinkdb_exporter" && cd dist && zip -9    rethinkdb_exporter-$CIRCLE_TAG.windows-amd64.zip   rethinkdb_exporter.exe && rm rethinkdb_exporter.exe && cd ..
gox -rebuild --osarch="windows/386"   -ldflags "$GO_LDFLAGS" -output "dist/rethinkdb_exporter" && cd dist && zip -9    rethinkdb_exporter-$CIRCLE_TAG.windows-386.zip     rethinkdb_exporter.exe && rm rethinkdb_exporter.exe && cd ..

echo "Upload to Github"
ghr -t $GITHUB_TOKEN -u $CIRCLE_PROJECT_USERNAME -r $CIRCLE_PROJECT_REPONAME --replace $CIRCLE_TAG dist/

docker version

gox --osarch="linux/386"   -ldflags "$GO_LDFLAGS" -output "dist/rethinkdb_exporter"

echo "Build Docker images"

echo "docker login"
docker login -u $DOCKER_USER -p $DOCKER_PASS
echo "docker login done"

docker build --rm=false -t "21zoo/rethinkdb_exporter:$CIRCLE_TAG" .
docker build --rm=false -t "21zoo/rethinkdb_exporter:latest" .

docker push "21zoo/rethinkdb_exporter:latest"
docker push "21zoo/rethinkdb_exporter:$CIRCLE_TAG"

docker build --rm=false -t "oliver006/rethinkdb_exporter:$CIRCLE_TAG" .
docker build --rm=false -t "oliver006/rethinkdb_exporter:latest" .
docker push "oliver006/rethinkdb_exporter:latest"
docker push "oliver006/rethinkdb_exporter:$CIRCLE_TAG"

echo "Done"
