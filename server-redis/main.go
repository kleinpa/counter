package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	pb "github.com/kleinpa/counter/api"
	"google.golang.org/grpc"
)

type counterServer struct {
	client *redis.Client

	pb.UnimplementedCounterServer
}

func NewCounterServer(client *redis.Client) *counterServer {
	return &counterServer{
		client: client,
	}
}

func (s *counterServer) Increment(ctx context.Context, req *pb.IncrementRequest) (*pb.IncrementReply, error) {
	i, err := s.client.Incr(ctx, req.GetId()).Result()
	if err != nil {
		return nil, err
	}
	go func(id string) {
		s.client.Publish(ctx, id, i)
	}(req.GetId())
	return &pb.IncrementReply{Value: int32(i)}, nil
}

func (s *counterServer) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetReply, error) {
	err := s.client.Set(ctx, req.GetId(), req.GetValue(), 0).Err()
	if err != nil {
		return nil, err
	}
	go func(id string, x int32) {
		s.client.Publish(ctx, id, x)
	}(req.GetId(), req.GetValue())
	log.Printf("set %v:%v", req.GetId(), req.GetValue())
	return &pb.SetReply{}, nil
}

func (s counterServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetReply, error) {
	i, err := s.client.Get(ctx, req.GetId()).Int64()
	if err != nil {
		return nil, err
	}
	return &pb.GetReply{Value: int32(i)}, nil
}

func (s counterServer) Watch(req *pb.WatchRequest, stream pb.Counter_WatchServer) error {
	pubsub := s.client.Subscribe(stream.Context(), req.GetId())
	ch := pubsub.Channel()

	for range ch {
		i, err := s.client.Get(stream.Context(), req.GetId()).Int64()
		if err != nil {
			return fmt.Errorf("weird error")
		}
		stream.Send(&pb.WatchReply{Value: int32(i)})
	}
	log.Printf("lol?")
	return fmt.Errorf("not implemented")
}

func startGrpcWebServer(grpcServer *grpc.Server, grpcWebAddress *string) {
	wrappedGrpc := grpcweb.WrapServer(grpcServer)
	httpServer := http.Server{
		Addr: *grpcWebAddress,
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			resp.Header().Add("Access-Control-Allow-Origin", req.Header.Get("Origin"))
			resp.Header().Set("Access-Control-Allow-Headers", "x-user-agent,content-type,x-grpc-web")
			if wrappedGrpc.IsGrpcWebRequest(req) {
				wrappedGrpc.ServeHTTP(resp, req)
			}
		}),
	}
	log.Printf("gRPC-Web listening on '%s'", *grpcWebAddress)

	httpServer.ListenAndServe()
}

func startGrpcServer(grpcServer *grpc.Server, grpcAddress *string) {
	lis, err := net.Listen("tcp", *grpcAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("gRPC listening on '%s'", *grpcAddress)
	grpcServer.Serve(lis)
}

func main() {

	grpcWebPort := os.Getenv("PORT")
	if grpcWebPort == "" {
		grpcWebPort = fmt.Sprintf("%v", pb.DefaultGRPCWebPort)
	}
	var grpcWebAddress = flag.String("grpc_web_address", fmt.Sprintf(":%s", grpcWebPort),
		"listen address for gRPC-Web service")

	grpcPort := fmt.Sprintf("%v", pb.DefaultGRPCPort)
	var grpcAddress = flag.String("grpc_address", fmt.Sprintf(":%s", grpcPort),
		"listen address for gRPC-Web service")

	var redisAddr = flag.String("redis_addr", "peterklein-counter.redis.cache.windows.net",
		"Address of redis server")
	var redisPassword = flag.String("redis_password", "",
		"Password of redis server")

	flag.Parse()

	rdb := redis.NewClient(&redis.Options{
		Addr:      *redisAddr,
		Password:  *redisPassword,
		TLSConfig: &tls.Config{},
		DB:        0, // use default DB
	})

	grpcServer := grpc.NewServer()
	server := NewCounterServer(rdb)
	pb.RegisterCounterServer(grpcServer, server)

	var wg sync.WaitGroup
	if *grpcWebAddress != "" {
		wg.Add(1)
		go func() {
			startGrpcWebServer(grpcServer, grpcWebAddress)
			wg.Done()
		}()
	}
	if *grpcAddress != "" {
		wg.Add(1)
		go func() {
			startGrpcServer(grpcServer, grpcAddress)
			wg.Done()
		}()
	}
	wg.Wait()
}
