package service

import (
	"context"
	"fmt"
	"time"

	"github.com/94peter/log"
	"github.com/94peter/storage"
	"github.com/94peter/storage/grpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func NewGcp(cfg *storage.Config) pb.GcpServiceServer {
	return &gcp{
		configMap: cfg.ConfMap,
		log:       cfg.Log,
	}
}

type gcp struct {
	pb.UnimplementedGcpServiceServer

	configMap storage.GcpConfigMap
	log       log.Logger
}

func getChannel(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.InvalidArgument, "can not get metadata")
	}
	mdChannel := md.Get("X-Channel")
	if len(mdChannel) == 0 {
		return "", status.Error(codes.InvalidArgument, "X-Channel not found")
	}
	return md.Get("X-Channel")[0], nil
}

func (gcp *gcp) getStorage(ctx context.Context, channel string) (storage.GcpStorage, error) {
	gcpConf := gcp.configMap.GetConfig(channel)
	if gcpConf == nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("channel not found [%s]", channel))
	}
	gcpStorage, err := gcpConf.NewStorage(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return gcpStorage, nil
}

// 取得下載連結
func (gcp *gcp) GetDownloadUrl(ctx context.Context, key *pb.ObjectKey) (*pb.Url, error) {
	channel, err := getChannel(ctx)
	if err != nil {
		return nil, err
	}
	gcpStorage, err := gcp.getStorage(ctx, channel)
	if err != nil {
		return nil, err
	}
	url, err := gcpStorage.GetDownloadUrl(key.Key)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	response := &pb.Url{
		Url:      url.Url,
		IsPublic: url.IsPublic,
	}

	if url.AccessToken != nil {
		response.Token = &pb.AccessToken{
			AccessToken:  url.AccessToken.AccessToken,
			RefreshToken: url.AccessToken.RefreshToken,
			TokenType:    url.AccessToken.TokenType,
			Expiry:       url.AccessToken.Expiry.Unix(),
		}
	}
	return response, nil
}

// 取得檔案
func (gcp *gcp) GetFile(ctx context.Context, key *pb.ObjectKey) (*pb.File, error) {
	channel, err := getChannel(ctx)
	if err != nil {
		return nil, err
	}
	gcpStorage, err := gcp.getStorage(ctx, channel)
	if err != nil {
		return nil, err
	}
	data, err := gcpStorage.Get(key.Key)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &pb.File{File: data}, nil
}

// 取得簽章
func (gcp *gcp) GetSignedUrl(ctx context.Context, req *pb.GetSignedUrlRequest) (*pb.Url, error) {
	channel, err := getChannel(ctx)
	if err != nil {
		return nil, err
	}
	gcpStorage, err := gcp.getStorage(ctx, channel)
	if err != nil {
		return nil, err
	}
	url, err := gcpStorage.SignedURL(req.Key, req.ContentType, time.Duration(req.ExpireSecs)*time.Second)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Url{Url: url}, nil
}

// 取得 AccessToken
func (gcp *gcp) GetAccessToken(ctx context.Context, empty *emptypb.Empty) (*pb.AccessToken, error) {
	channel, err := getChannel(ctx)
	if err != nil {
		return nil, err
	}
	gcpStorage, err := gcp.getStorage(ctx, channel)
	if err != nil {
		return nil, err
	}
	token, err := gcpStorage.GetAccessToken()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.AccessToken{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry.Unix(),
	}, nil
}

func (gcp *gcp) SaveFile(ctx context.Context, req *pb.SaveFileRequest) (*pb.Url, error) {
	channel, err := getChannel(ctx)
	if err != nil {
		return nil, err
	}
	gcpStorage, err := gcp.getStorage(ctx, channel)
	if err != nil {
		return nil, err
	}
	path, err := gcpStorage.Save(req.Key, req.File)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Url{Url: path}, nil
}

// 刪除
func (gcp *gcp) Delete(ctx context.Context, key *pb.ObjectKey) (*emptypb.Empty, error) {
	channel, err := getChannel(ctx)
	if err != nil {
		return nil, err
	}
	gcpStorage, err := gcp.getStorage(ctx, channel)
	if err != nil {
		return nil, err
	}
	err = gcpStorage.Delete(key.Key)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

// 檢查檔案是否存在
func (gcp *gcp) Exist(ctx context.Context, key *pb.ObjectKey) (*pb.ExistResponse, error) {
	channel, err := getChannel(ctx)
	if err != nil {
		return nil, err
	}
	gcpStorage, err := gcp.getStorage(ctx, channel)
	if err != nil {
		return nil, err
	}
	exist, err := gcpStorage.FileExist(key.Key)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.ExistResponse{Exist: exist}, nil
}

// 列出
func (gcp *gcp) List(ctx context.Context, dir *pb.Dir) (*pb.ListResponse, error) {
	channel, err := getChannel(ctx)
	if err != nil {
		return nil, err
	}
	gcpStorage, err := gcp.getStorage(ctx, channel)
	if err != nil {
		return nil, err
	}
	files, err := gcpStorage.List(dir.Path)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.ListResponse{Files: files}, nil
}
