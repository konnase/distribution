FROM debian

ENV DISTRIBUTION_DIR /go/src/github.com/docker/distribution
ENV DOCKER_BUILDTAGS include_oss include_gcs

#Install make and git and glusterfs-api-devel
RUN apt-get update
RUN apt-get install -y wget gcc automake autoconf libtool make git glusterfs-client pkg-config

RUN glusterfs -V

#Install golang 1.8
RUN wget http://www.golangtc.com/static/go/1.8/go1.8.linux-amd64.tar.gz
RUN tar -xzf go1.8.linux-amd64.tar.gz -C /usr/local
ENV PATH $PATH:/usr/local/go/bin
ENV GOROOT /usr/local/go
ENV GOBIN /go/bin
ENV GOPATH /go
RUN go version


WORKDIR $DISTRIBUTION_DIR
COPY . $DISTRIBUTION_DIR
COPY cmd/registry/config-dev.yml /etc/docker/registry/config.yml


RUN make PREFIX=/go clean binaries
RUN mkdir /mnt/gluster

#VOLUME ["/mnt/gluster"]
EXPOSE 5000
ENTRYPOINT ["/go/bin/registry"]
CMD ["serve", "/etc/registry/config.yml"]
