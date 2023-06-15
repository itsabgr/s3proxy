FROM golang:1.20
WORKDIR /src
COPY vendor vendor
COPY go.mod go.sum ./
COPY *.go ./
COPY s3proxy.yaml /etc/s3proxy.yaml
RUN go test ./...
RUN go install .
EXPOSE 80 443
CMD s3proxy -h