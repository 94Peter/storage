package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	googstorage "cloud.google.com/go/storage"
	"github.com/94peter/storage/grpc/pb"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GrpcGcpStorage interface {
	GcpStorage
	Close()
}

func NewGrpcGcpStorage(ctx context.Context, address string, channel string) (GrpcGcpStorage, error) {
	md := metadata.New(map[string]string{"X-Channel": channel})
	ctx = metadata.NewOutgoingContext(ctx, md)
	conn, err := getClient(address)
	if err != nil {
		return nil, err
	}
	return &grpcStorage{
		ctx:  ctx,
		conn: conn,
	}, nil
}

func getClient(address string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(address,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("address [%s] error: %s", address, err.Error())
	}
	return conn, nil
}

type grpcStorage struct {
	ctx  context.Context
	conn *grpc.ClientConn
}

func (s *grpcStorage) Close() {
	s.conn.Close()
}

func (gcp *grpcStorage) Save(filePath string, file []byte) (string, error) {
	return gcp.Write(filePath, func(w io.Writer) error {
		_, err := w.Write(file)
		return err
	})
}

func (gcp *grpcStorage) SaveByReader(fp string, reader io.Reader) (string, error) {
	return gcp.Write(fp, func(w io.Writer) error {
		_, err := io.Copy(w, reader)
		return err
	})
}

func (gcp *grpcStorage) Write(key string, writeData func(w io.Writer) error) (path string, err error) {
	clt := pb.NewGcpServiceClient(gcp.conn)
	buf := new(bytes.Buffer)
	if err = writeData(buf); err != nil {
		err = fmt.Errorf("write file error: %s", err.Error())
		return
	}
	url, err := clt.SaveFile(gcp.ctx, &pb.SaveFileRequest{
		Key:  key,
		File: buf.Bytes(),
	})
	if err != nil {
		return
	}
	path = url.Url
	return
}

func (gcp *grpcStorage) Delete(key string) error {
	clt := pb.NewGcpServiceClient(gcp.conn)
	_, err := clt.Delete(gcp.ctx, &pb.ObjectKey{Key: key})
	return err
}

func (gcp *grpcStorage) OpenFile(key string) (io.Reader, error) {
	clt := pb.NewGcpServiceClient(gcp.conn)
	file, err := clt.GetFile(gcp.ctx, &pb.ObjectKey{Key: key})
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(file.File), nil
}

func (gcp *grpcStorage) GetAttr(key string) (*googstorage.ObjectAttrs, error) {
	panic("not implemented")
}

func (gcp *grpcStorage) FileExist(fp string) (bool, error) {
	clt := pb.NewGcpServiceClient(gcp.conn)
	rsp, err := clt.Exist(gcp.ctx, &pb.ObjectKey{Key: fp})
	if err != nil {
		return false, err
	}
	return rsp.Exist, nil
}

func (gcp *grpcStorage) GetDownloadUrl(key string) (myurl string, err error) {
	clt := pb.NewGcpServiceClient(gcp.conn)
	url, err := clt.GetDownloadUrl(gcp.ctx, &pb.ObjectKey{Key: key})
	if err != nil {
		return
	}
	myurl = url.Url
	return
}

func (gcp *grpcStorage) SignedURL(key string, contentType string, expSecs time.Duration) (string, error) {
	clt := pb.NewGcpServiceClient(gcp.conn)
	url, err := clt.GetSignedUrl(gcp.ctx, &pb.GetSignedUrlRequest{Key: key, ContentType: contentType, ExpireSecs: uint32(expSecs / time.Second)})
	if err != nil {
		return "", err
	}
	return url.Url, nil
}

func (gcp *grpcStorage) GetAccessToken() (*oauth2.Token, error) {
	clt := pb.NewGcpServiceClient(gcp.conn)
	token, err := clt.GetAccessToken(gcp.ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	return &oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       time.Unix(token.Expiry, 0),
	}, nil
}

func (gcp *grpcStorage) Get(key string) ([]byte, error) {
	clt := pb.NewGcpServiceClient(gcp.conn)
	file, err := clt.GetFile(gcp.ctx, &pb.ObjectKey{Key: key})
	if err != nil {
		return nil, err
	}
	return file.File, nil
}

func (gcp *grpcStorage) List(dir string) ([]string, error) {
	clt := pb.NewGcpServiceClient(gcp.conn)
	rsp, err := clt.List(gcp.ctx, &pb.Dir{Path: dir})
	if err != nil {
		return nil, err
	}
	return rsp.Files, nil
}
