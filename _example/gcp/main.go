package main

import (
	"context"
	"fmt"
	"time"

	"github.com/94peter/storage"
)

func main() {
	gcpConf := &storage.GcpConf{
		CredentialsFile: "./serviceAccountKey.json",
	}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	sto, err := gcpConf.NewStorage(ctx, "private_in_volunteer")
	if err != nil {
		fmt.Println(err)
		return
	}
	path, err := sto.Save("product/hello.txt", []byte("hello world"))
	fmt.Println(path, err)

	list, err := sto.List("product/")
	fmt.Println(list, err)

	exist, err := sto.FileExist("product/hello.txt")
	fmt.Println(exist, err)

	exist, err = sto.FileExist("product/not_exist.txt")
	fmt.Println(exist, err)

	data, err := sto.Get("product/hello.txt")
	fmt.Println(string(data), err)

	err = sto.Delete("product/hello.txt")
	fmt.Println(err)

	fmt.Println(sto.GetDownloadUrl("product/hello.txt"))
}
