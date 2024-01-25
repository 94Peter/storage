package main

import (
	"context"
	"fmt"

	mygrpc "github.com/94peter/storage/grpc"
)

func main() {
	clt := mygrpc.NewGcpGrpcClient(context.Background(), "127.0.0.1:1080")
	fmt.Println(clt.List("default", "mail"))
}
