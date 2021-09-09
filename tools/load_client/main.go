package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/kleinpa/counter/api"
	"google.golang.org/grpc"
)

var Counters = 100
var Requests = 100000

func main() {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	conn, err := grpc.Dial(fmt.Sprintf("localhost:%v", pb.DefaultGRPCPort), opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewCounterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	counters := []string{}

	for i := 0; i <= Counters; i++ {
		id := uuid.New().String()
		_, err := client.Set(ctx, &pb.SetRequest{Id: id, Value: 0})
		if err != nil {
			log.Fatalf("could not set counter %v id %q", i, id)
		}
		log.Printf("counter %v id %q", i, id)
		counters = append(counters, id)

		go func(i int) {
			stream, err := client.Watch(ctx, &pb.WatchRequest{Id: id})
			if err != nil {
				log.Fatalf("watch failed: %v", err)
			}
			for {
				_, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					break
				}
			}
		}(i)
	}

	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < Requests; i++ {
		wg.Add(1)
		go func() {
			id := counters[rand.Int()%len(counters)]
			_, err := client.Increment(ctx, &pb.IncrementRequest{Id: id, Value: 1})
			if err != nil {
				log.Fatalf("increment failed: %v", err)
			}
			defer wg.Done()
		}()
	}
	wg.Wait()
	duration := time.Since(start)

	total := int32(0)
	for _, id := range counters {
		res, err := client.Get(ctx, &pb.GetRequest{Id: id})
		if err != nil {
			log.Fatalf("get failed: %v", err)
		}
		fmt.Printf("counter %v: %v\n", id, res.Value)
		total = total + res.Value
	}
	fmt.Printf("requests: %v counts: %v duration: %.2f s\n", Requests, total, duration.Seconds())
	fmt.Printf("qps: %.1f / s\n", float64(Requests)/duration.Seconds())
}
