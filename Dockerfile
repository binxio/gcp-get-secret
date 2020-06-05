FROM golang:1.14
WORKDIR /go

ADD . /go/src/github.com/binxio/gcp-get-secret

RUN go get github.com/binxio/gcp-get-secret
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' github.com/binxio/gcp-get-secret

FROM php:7.4-apache
COPY --from=0 /go/gcp-get-secret /

ADD index.php /var/www/html
ADD secret-docker-php-entrypoint /

ENTRYPOINT [ "/secret-docker-php-entrypoint" ]
