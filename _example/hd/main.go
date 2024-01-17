package main

import (
	"fmt"

	"github.com/94peter/storage"
)

func main() {
	hdstorage := storage.NewHdStorage("./test")
	key, err := hdstorage.Save("hello.txt", []byte("hello world"))
	fmt.Println(key, err)
}
