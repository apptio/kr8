FROM golang:1.10.4-alpine3.8

WORKDIR /go/src/github.com/apptio/kr8

COPY . /go/src/github.com/apptio/kr8

ARG VERSION

RUN apk add --no-cache git curl \
    && curl https://glide.sh/get | sh \
    && ls . \
    && glide i \
    && go build -o kr8 -ldflags "-X main.version=${VERSION}"

ENTRYPOINT ["./kr8"]
