package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/94peter/log"
	"github.com/94peter/microservice"
	"github.com/94peter/microservice/di"
	"github.com/94peter/microservice/grpc_tool"
	healthpb "github.com/94peter/microservice/grpc_tool/health/pb"
	"github.com/94peter/storage"
	"github.com/94peter/storage/container/service"
	"github.com/94peter/storage/grpc/pb"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

var (
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

	exePath, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	envFile := exePath + "/.env"
	if fileExists(envFile) {
		err := godotenv.Load(envFile)
		if err != nil {
			panic("load .env file fail")
		}
	}

	modelCfg, err := storage.GetConfigFromEnv()
	if err != nil {
		panic(err)
	}

	microService, err := microservice.New(modelCfg, &mydi{})
	if err != nil {
		panic(err)
	}

	service := newService(microService)
	microservice.RunService(
		service.runGrpc,
	)

}

func newService(microService microservice.MicroService[*storage.Config, *mydi]) *myservice {
	return &myservice{
		MicroService: microService,
	}
}

type myservice struct {
	microservice.MicroService[*storage.Config, *mydi]
}

func (s *myservice) runGrpc(ctx context.Context) {
	cfg, err := s.NewCfg("grpc")
	if err != nil {
		panic(err)
	}

	grpcCfg, err := grpc_tool.GetConfigFromEnv()
	if err != nil {
		panic(err)
	}
	grpcCfg.SetRegisterServiceFunc(func(s *grpc.Server) {
		pb.RegisterGcpServiceServer(s, service.NewGcp(cfg))
		healthpb.RegisterHealthServer(s, service.NewHealthService())
	})

	grpcCfg.Logger = cfg.Log

	grpc_tool.RunGrpcServ(ctx, grpcCfg)

}

type mydi struct {
	di.CommonServiceDI

	*log.LoggerConf `yaml:"log"`
}

func (d *mydi) IsConfEmpty() error {
	if log.EnvHasFluentd() && (d.LoggerConf == nil || d.FluentLog == nil) {
		return errors.New("log.FluentLog no set")
	}
	return nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
