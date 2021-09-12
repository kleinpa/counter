# Start by building the application.
FROM golang:1.17.1-buster as build

RUN apt-get update && apt-get install --no-install-recommends -y \
    unzip \
 && apt-get autoremove -y && apt-get clean -y && rm -rf /var/lib/apt/lists/* /tmp/library-scripts

RUN curl -sL https://github.com/protocolbuffers/protobuf/releases/download/v3.18.0-rc2/protoc-3.18.0-rc-2-linux-x86_64.zip -o /tmp/protoc-3.18.0-rc-2-linux-x86_64.zip \
  && mkdir /tmp/protoc && unzip /tmp/protoc-3.18.0-rc-2-linux-x86_64.zip -d /tmp/protoc \
  && mv /tmp/protoc/bin/protoc /usr/local/bin/protoc \
  && mv /tmp/protoc/include /usr/local/include

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26 \
 && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1

WORKDIR /go/src/app
ADD . /go/src/app/

RUN go generate -v ./...
RUN go get -d -v ./...
RUN go build -o /go/bin ./...

FROM gcr.io/distroless/base-debian10
COPY --from=build /go/bin/server /server
CMD ["/server"]
