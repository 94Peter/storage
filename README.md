# Storage

專案提供檔案存取library，支援google storage及本地檔案存取

# Example

## 本地檔案存
```go
func main() {
	hdstorage := storage.NewHdStorage("./test")
	key, err := hdstorage.Save("hello.txt", []byte("hello world"))
	fmt.Println(key, err)
}
```

## gcp檔案存取
```go
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
```

# container服務
[README](container/README.md)