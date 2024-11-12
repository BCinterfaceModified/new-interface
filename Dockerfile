ARG GO_VER=1.21
ARG ALPINE_VER=3.19
ARG PORT

FROM alpine:${ALPINE_VER} AS peer-base
RUN apk add --no-cache tzdata

RUN echo 'hosts: files dns' > /etc/nsswitch.conf

FROM golang:${GO_VER}-alpine${ALPINE_VER} AS golang
RUN apk add --no-cache \
	bash \
	gcc \
	git \
	make \
	musl-dev
ADD . $GOPATH/src/github.com/new_interface
WORKDIR $GOPATH/src/github.com/new_interface

FROM golang AS peer
RUN go build -o server

FROM peer-base
COPY --from=peer /go/src/github.com/new_interface /usr/local/bin
EXPOSE 50051
CMD  ["server", "-REDIS_ADDR=${REDIS_ADDR}", "-MONGO_ADDR=${MONGO_ADDR}", "-GRPC_PORT=${GRPC_PORT}"]