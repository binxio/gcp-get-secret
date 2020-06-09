FROM golang:1.14
WORKDIR /go

ADD . /go/src/github.com/binxio/gcp-get-secret

RUN go get github.com/binxio/gcp-get-secret
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' github.com/binxio/gcp-get-secret

FROM 		scratch
COPY --from=0		/go/gcp-get-secret /