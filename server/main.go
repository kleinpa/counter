package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	pb "github.com/kleinpa/counter/api"
	"google.golang.org/grpc"
)

type counter struct {
	value        int32
	modified     time.Time
	listeners    []*chan int32
	listeners_mu sync.RWMutex
}

func NewCounter() *counter {
	return &counter{
		listeners: make([]*chan int32, 0),
	}
}

func (c *counter) Watch() *chan int32 {
	c.listeners_mu.Lock()
	defer c.listeners_mu.Unlock()

	ch := make(chan int32, 0)
	c.listeners = append(c.listeners, &ch)
	return &ch
}

func (c *counter) EndWatch(ch *chan int32) {
	c.listeners_mu.Lock()
	defer c.listeners_mu.Unlock()

	for i := 0; i < len(c.listeners); i++ {
		if c.listeners[i] == ch {
			c.listeners = append(c.listeners[:i], c.listeners[i+1:]...)
		}
	}

	close(*ch)
}

func (c *counter) NotifyListeners(v int32) {
	c.listeners_mu.RLock()
	defer c.listeners_mu.RUnlock()

	// HERE BE RACE CONDITIONS
	// what happens when a client has disconnected but the EndWatch has not been processed?
	for _, listener := range c.listeners {
		*listener <- v
	}
}

type counterServer struct {
	counters map[string]*counter

	pb.UnimplementedCounterServer
}

func NewCounterServer() *counterServer {
	return &counterServer{
		counters: make(map[string]*counter),
	}
}

func (s *counterServer) getCounter(id string) (*counter, error) {
	if _, ok := s.counters[id]; !ok {
		s.counters[id] = &counter{}
	}
	c, _ := s.counters[id]
	return c, nil
}

func (s *counterServer) Increment(ctx context.Context, req *pb.IncrementRequest) (*pb.IncrementReply, error) {
	counter, err := s.getCounter(req.Id)
	if err != nil {
		return nil, err
	}
	v := atomic.AddInt32(&counter.value, req.Value)
	counter.modified = time.Now()
	go counter.NotifyListeners(v)
	return &pb.IncrementReply{Value: v}, nil
}

func (s *counterServer) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetReply, error) {
	counter, err := s.getCounter(req.Id)
	if err != nil {
		return nil, err
	}
	atomic.StoreInt32(&counter.value, req.Value)
	counter.modified = time.Now()
	go counter.NotifyListeners(req.Value)
	return &pb.SetReply{}, nil
}

func (s counterServer) Get(ctz context.Context, req *pb.GetRequest) (*pb.GetReply, error) {
	counter, ok := s.counters[req.Id]
	if !ok {
		return nil, fmt.Errorf("no counter with id %q", req.Id)
	}
	return &pb.GetReply{Value: atomic.LoadInt32(&counter.value)}, nil
}

func (s counterServer) Watch(req *pb.WatchRequest, stream pb.Counter_WatchServer) error {
	counter, ok := s.counters[req.Id]
	if !ok {
		return fmt.Errorf("no counter with id %q", req.Id)
	}
	stream.Send(&pb.WatchReply{Value: atomic.LoadInt32(&counter.value)})
	ch := counter.Watch()
	defer counter.EndWatch(ch)
	for i := range *ch {
		if err := stream.Send(&pb.WatchReply{Value: i}); err != nil {
			return err
		}
	}
	return nil
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

	flag.Parse()

	grpcServer := grpc.NewServer()
	server := NewCounterServer()
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
