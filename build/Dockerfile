FROM centos:latest

# Install rpm requirements
RUN dnf -y install git curl tar rpm-build rpmdevtools

# Install Golang
ARG GOLANG_VERSION=1.17
ARG GOLANG_ARCH=amd64
WORKDIR /usr/local
RUN curl -L https://dl.google.com/go/go${GOLANG_VERSION}.linux-${GOLANG_ARCH}.tar.gz | tar -xzf -
RUN ln -s /usr/local/go/bin/go /usr/bin/go
RUN ln -s /usr/local/go/bin/gofmt /usr/bin/gofmt

# Build RPM
WORKDIR /root/rpmbuild
ADD uptimed.spec .
RUN mkdir SOURCES
COPY uptimed-*.tar.gz SOURCES/

VOLUME /root/rpmbuild/RPMS

ENTRYPOINT [ "rpmbuild", "-ba", "uptimed.spec" ]
