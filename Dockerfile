FROM node:8.12-alpine as node

COPY web/package.json ./
COPY web/yarn.lock ./

RUN yarn install

ADD ./web ./

RUN \
    node_modules/.bin/node-sass static/sass/application.scss static/sass/application.css && \
    node_modules/.bin/postcss static/sass/application.css -o static/sass/application.css

FROM golang:1.11.4-alpine3.7 as builder

ENV GOPATH /go
ENV CGO_ENABLED 0

WORKDIR /go/src/github.com/syaiful6/thatique

ADD . /go/src/github.com/syaiful6/thatique/
COPY --from=node /static/sass/application.css /go/src/github.com/syaiful6/thatique/assets/static/css/application.css

RUN \
    apk add --no-cache git && \
    go get -u github.com/jteeuwen/go-bindata/... && \
    go-bindata -o assets/assets.go -pkg assets assets/... && \
    go install

FROM alpine:3.7

RUN apk add tini

ENV THATIQUE_CONFIGURATION_PATH="/data/thatique-config.yml"

COPY --from=builder /go/bin/thatique /usr/bin/thatique

VOLUME ["/data"]

ENTRYPOINT ["tini", "--"]

CMD thatique serve
