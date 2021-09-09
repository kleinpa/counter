//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative counter.proto
package api

var DefaultGRPCPort = 8409
var DefaultGRPCWebPort = 8408
