FROM golang:1.14

WORKDIR /gcp-get-secret
ADD . /gcp-get-secret
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o gcp-get-secret -ldflags '-extldflags "-static"' .

FROM scratch
COPY --from=0 /gcp-get-secret/gcp-get-secret /
