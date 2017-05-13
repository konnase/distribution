FROM gluster/gluster-centos

ENV DISTRIBUTION_DIR /go/src/github.com/docker/distribution
ENV DOCKER_BUILDTAGS include_oss include_gcs

#Install golang 1.8
RUN wget http://www.golangtc.com/static/go/1.8/go1.8.linux-amd64.tar.gz
RUN tar -xzf go1.8.linux-amd64.tar.gz -C /usr/local
ENV PATH $PATH:/usr/local/go/bin
ENV GOROOT /usr/local/go
ENV GOBIN /go/bin
ENV GOPATH /go
RUN go version

#Install make and git and glusterfs-api-devel
RUN yum -y install gcc automake autoconf libtool make
RUN yum -y install git
RUN yum -y install glusterfs-api-devel

RUN glusterfs -V

WORKDIR $DISTRIBUTION_DIR
COPY . $DISTRIBUTION_DIR
COPY cmd/registry/config-dev.yml /etc/docker/registry/config.yml


RUN make PREFIX=/go clean binaries

#VOLUME ["/mnt/gluster"]
EXPOSE 5000
#ENTRYPOINT ["/go/bin/registry"]
CMD ["serve", "/etc/docker/registry/config.yml"]
