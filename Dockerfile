FROM golang:1.14
WORKDIR /go
#RUN go get -u google.golang.org/genproto/googleapis/cloud/secretmanager/v1
#RUN go get -u cloud.google.com/go/secretmanager/apiv1

ADD . /go/src/github.com/binxio/gcp-get-secret

RUN go get github.com/binxio/gcp-get-secret
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' github.com/binxio/gcp-get-secret

FROM alpine
COPY --from=0 /go/gcp-get-secret /