package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/94peter/di"
	"github.com/94peter/log"
	"github.com/94peter/storage"
	"github.com/94peter/storage/container/service"
	"github.com/94peter/storage/grpc/pb"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	envMode = flag.String("em", "local", "local, container")

	v         = flag.Bool("v", false, "version")
	Version   = "1.0.0"
	BuildTime = time.Now().Local().GoString()
)

func main() {
	flag.Parse()

	if *v {
		fmt.Println("Version: " + Version)
		fmt.Println("Build Time: " + BuildTime)
		return
	}

	if *envMode == "local" {
		err := godotenv.Load(".env")
		if err != nil {
			fmt.Println("No .env file")
		}
	}

	confPath := os.Getenv("CONF_PATH")
	serviceName := os.Getenv(("SERVICE"))
	hdstore := storage.NewHdStorage(confPath)
	mydi := &mydi{
		serviceName: serviceName,
	}
	confByte, err := hdstore.Get("config.yml")
	if err != nil {
		panic(err)
	}
	err = di.InitConfByByte(confByte, mydi)
	if err != nil {
		panic(err)
	}
	if err = mydi.IsConfEmpty(); err != nil {
		panic(err)
	}

	for {
		runGrpc(mydi)
		time.Sleep(time.Second)
	}

}

func runGrpc(mydi *mydi) {
	port := ":" + os.Getenv("GRPC_PORT")

	confPath := os.Getenv("GCP_CONF_MAP_PATH")
	configMap, err := storage.LoadGcpConfigMap(confPath)
	if err != nil {
		panic("load gcp conf map fail: " + err.Error())
	}
	l, err := mydi.NewLogger(mydi.GetService(), "grpc")
	if err != nil {
		panic(err)
	}
	lis, err := net.Listen("tcp", port)
	if err != nil {
		l.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	reflection.Register(s)
	pb.RegisterGcpServiceServer(s, service.NewGcp(configMap, l))

	l.Infof("app gRPC server is running [%s].", port)
	if err := s.Serve(lis); err != nil {
		l.Fatalf("failed to serve: %v", err)
	}
}

type mydi struct {
	serviceName string

	*log.LoggerConf `yaml:"log"`
}

func (d *mydi) IsConfEmpty() error {
	if log.EnvHasFluentd() && (d.LoggerConf == nil || d.FluentLog == nil) {
		return errors.New("log.FluentLog no set")
	}
	return nil
}

func (d *mydi) GetService() string {
	return d.serviceName
}
