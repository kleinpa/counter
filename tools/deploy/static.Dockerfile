FROM docker.io/library/node:13.12.0-buster-slim AS build

RUN apt-get update && apt-get install --no-install-recommends -y \
    unzip \
    ca-certificates \
    curl \
 && apt-get autoremove -y && apt-get clean -y && rm -rf /var/lib/apt/lists/* /tmp/library-scripts

RUN curl -sL https://github.com/protocolbuffers/protobuf/releases/download/v3.18.0-rc2/protoc-3.18.0-rc-2-linux-x86_64.zip -o /tmp/protoc-3.18.0-rc-2-linux-x86_64.zip \
  && mkdir /tmp/protoc && unzip /tmp/protoc-3.18.0-rc-2-linux-x86_64.zip -d /tmp/protoc \
  && mv /tmp/protoc/bin/protoc /usr/local/bin/protoc \
  && mv /tmp/protoc/include /usr/local/include \
  && curl -sL https://github.com/grpc/grpc-web/releases/download/1.2.1/protoc-gen-grpc-web-1.2.1-linux-x86_64 -o curl -sL https://github.com/grpc/grpc-web/releases/download/1.2.1/protoc-gen-grpc-web-1.2.1-linux-x86_64 -o /usr/local/bin/protoc-gen-grpc-web \
  && chmod +x /usr/local/bin/protoc-gen-grpc-web

WORKDIR /build
COPY package.json package-lock.json ./
RUN npm ci

COPY . ./
RUN protoc -Iapi --js_out=import_style=commonjs:src --grpc-web_out=import_style=typescript,mode=grpcwebtext:src api/counter.proto
RUN npm run build

FROM docker.io/library/nginx:1.21.3-alpine
COPY tools/deploy/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /build/build /usr/share/nginx/html
