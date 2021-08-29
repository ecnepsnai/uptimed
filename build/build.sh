#!/bin/bash
set -e

UPTIMED_VERSION=${1:?Version required}

cd ../
rm -rf artifacts
mkdir -p artifacts
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w"
mv uptimed artifacts/uptimed_linux_amd64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w"
mv uptimed artifacts/uptimed_linux_arm64

rm -rf uptimed-${UPTIMED_VERSION}
mkdir uptimed-${UPTIMED_VERSION}
cp *.go go.* build/uptimed.service build/uptimed.env uptimed-${UPTIMED_VERSION}
tar -czf uptimed-${UPTIMED_VERSION}.tar.gz uptimed-${UPTIMED_VERSION}
rm -rf uptimed-${UPTIMED_VERSION}
rm -f build/uptimed-${UPTIMED_VERSION}.tar.gz
mv uptimed-${UPTIMED_VERSION}.tar.gz build/

cd build
perl -pi -e "s,##VERSION##,${UPTIMED_VERSION},g" uptimed.spec

GOLANG_ARCH="amd64"
if [[ $(uname -m) == 'aarch64' ]]; then
    GOLANG_ARCH="arm64"
fi

podman build --build-arg UPTIMED_VERSION=${UPTIMED_VERSION} --build-arg GOLANG_ARCH=${GOLANG_ARCH} -t uptimed_build .
rm -rf rpms
mkdir -p rpms
podman run --user root -v $(readlink -f rpms):/root/rpmbuild/RPMS:Z uptimed_build
cp rpms/*/*.rpm ../artifacts

