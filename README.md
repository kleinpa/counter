Generate JavaScript/TypeScript gRPC files:

```sh
$ protoc -Iapi --js_out=import_style=commonjs:src --grpc-web_out=import_style=typescript,mode=grpcwebtext:src api/counter.proto
```
