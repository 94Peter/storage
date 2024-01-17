package storage

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func NewHdStorage(path string) Storage {
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}
	return &hd{Path: path}
}

type hd struct {
	Path string
}

func (hd *hd) getAbsFilePath(filePath string) string {
	absFilePath, _ := filepath.Abs(hd.Path + filePath)
	return absFilePath
}

func (hd *hd) Save(fp string, file []byte) (string, error) {
	absFilePath := hd.getAbsFilePath(fp)
	err := hd.mkdir(absFilePath)
	if err != nil {
		return "", err
	}
	return absFilePath, os.WriteFile(absFilePath, file, 0644)
}

func (hd *hd) SaveByReader(fp string, reader io.Reader) (string, error) {
	absFilePath := hd.getAbsFilePath(fp)
	err := hd.mkdir(absFilePath)
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(absFilePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = io.Copy(f, reader)
	if err != nil {
		return "", err
	}
	return absFilePath, nil
}

func (hd *hd) Delete(filePath string) error {
	absFilePath := hd.getAbsFilePath(filePath)
	exist, err := fileExist(absFilePath)
	if err != nil {
		return err
	}
	if !exist {
		return errors.New("file not exist: " + absFilePath)
	}
	return os.Remove(absFilePath)
}

func (hd *hd) Get(fp string) ([]byte, error) {
	absFilePath := hd.getAbsFilePath(fp)
	return os.ReadFile(absFilePath)
}

func (hd *hd) mkdir(absPath string) error {
	dir := filepath.Dir(absPath)
	exist, _ := fileExist(dir)

	if !exist {
		err := os.MkdirAll(dir, 0766)
		if err != nil {
			return err
		}
	}
	return nil
}

func (hd *hd) FileExist(fp string) (bool, error) {
	absFilePath := hd.getAbsFilePath(fp)
	exist, err := fileExist(absFilePath)
	if err != nil {
		return false, err
	}
	return exist, nil
}

func (hd *hd) List(dir string) ([]string, error) {
	absDir := hd.getAbsFilePath(dir)
	files, err := os.ReadDir(absDir)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, f := range files {
		if f.IsDir() {
			result = append(result, strAppend(dir, f.Name(), "/"))
		} else {
			result = append(result, strAppend(dir, f.Name()))
		}

	}
	return result, nil
}

func fileExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func strAppend(strs ...string) string {
	var buffer bytes.Buffer
	for _, str := range strs {
		buffer.WriteString(str)
	}
	return buffer.String()
}
