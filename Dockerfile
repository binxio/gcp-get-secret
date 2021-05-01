FROM alpine:3 as ca
RUN apk add --no-cache ca-certificates


FROM golang:1.16 as go

WORKDIR /gcp-get-secret
ADD . /gcp-get-secret
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o gcp-get-secret -ldflags '-extldflags "-static"' .

FROM scratch
COPY --from=ca /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=go /gcp-get-secret/gcp-get-secret /

ENTRYPOINT [ "/gcp-get-secret" ]
