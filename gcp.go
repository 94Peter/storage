package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/iam"
	googstorage "cloud.google.com/go/storage"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

type Perm string

const (
	PermPublic  = Perm("public")
	PermPrivate = Perm("private")
	PermTmp     = Perm("tmp")
)

type GcpDI interface {
	NewStorage(ctx context.Context, bucket string) (GcpStorage, error)
}

type GcpStorage interface {
	Storage
	GetAttr(key string) (*googstorage.ObjectAttrs, error)
	GetDownloadUrl(key string) (myurl *DownloadUrl, err error)
	Write(key string, writeData func(w io.Writer) error) (path string, err error)
	OpenFile(key string) (io.Reader, error)
	SignedURL(key string, contentType string, expDuration time.Duration) (url string, err error)
	GetAccessToken() (*oauth2.Token, error)
}

type GcpConf struct {
	CredentialsFile string `yaml:"credentailsFile"`
	CredentailsUrl  string `yaml:"credentailsUrl"`
	Bucket          string `yaml:"bucket"`
}

func downloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func (gcp *GcpConf) NewStorage(ctx context.Context) (GcpStorage, error) {
	if gcp.CredentailsUrl != "" {
		filePath := fmt.Sprintf("/tmp/%s.json", filenameEncode(gcp.CredentailsUrl))
		if !fileExists(filePath) {
			err := downloadFile(filePath, gcp.CredentailsUrl)
			if err != nil {
				return nil, err
			}
		}
		gcp.CredentialsFile = filePath
	}
	jsonKey, err := os.ReadFile(gcp.CredentialsFile)
	if err != nil {
		return nil, err
	}
	credentails, err := google.CredentialsFromJSON(context.Background(), jsonKey, googstorage.ScopeFullControl)
	if err != nil {
		return nil, err
	}

	return &storageImpl{
		ctx:         ctx,
		bucket:      gcp.Bucket,
		GcpConf:     gcp,
		jsonData:    jsonKey,
		credentials: credentails,
	}, nil
}

type storageImpl struct {
	ctx context.Context
	*GcpConf
	bucket      string
	jsonData    []byte
	credentials *google.Credentials
}

func (gcp *storageImpl) getClient() (*googstorage.Client, error) {
	return googstorage.NewClient(gcp.ctx, option.WithCredentials(gcp.credentials))
}

func (gcp *storageImpl) Save(filePath string, file []byte) (string, error) {
	return gcp.Write(filePath, func(w io.Writer) error {
		_, err := w.Write(file)
		return err
	})
}

func (gcp *storageImpl) SaveByReader(fp string, reader io.Reader) (string, error) {
	return gcp.Write(fp, func(w io.Writer) error {
		_, err := io.Copy(w, reader)
		return err
	})
}

func (gcp *storageImpl) Write(key string, writeData func(w io.Writer) error) (path string, err error) {
	client, err := gcp.getClient()
	if err != nil {
		err = fmt.Errorf("storage.NewClient: %v", err)
		return
	}
	defer client.Close()

	wc := client.Bucket(gcp.bucket).Object(key).NewWriter(gcp.ctx)
	if err = writeData(wc); err != nil {
		err = fmt.Errorf("write file error: %s", err.Error())
		return
	}
	if err = wc.Close(); err != nil {
		err = fmt.Errorf("createFile: unable to close bucket %q, file %q: %v", gcp.bucket, key, err)
		return
	}
	path = wc.Attrs().Name
	return
}

func (gcp *storageImpl) Delete(key string) error {
	client, err := gcp.getClient()
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	if err := client.Bucket(gcp.bucket).Object(key).Delete(gcp.ctx); err != nil {
		return fmt.Errorf("delete: unable to delete object bucket %q, file %q: %v", gcp.bucket, key, err)
	}

	return nil
}

func (gcp *storageImpl) OpenFile(key string) (io.Reader, error) {
	data, err := gcp.Get(key)
	if err != nil {
		return nil, fmt.Errorf("os.ReadAll fail: %v", err)
	}
	return bytes.NewReader(data), nil
}

func (gcp *storageImpl) GetAttr(key string) (*googstorage.ObjectAttrs, error) {
	client, err := gcp.getClient()
	if err != nil {
		err = fmt.Errorf("storage.NewClient: %v", err)
		return nil, err
	}
	defer client.Close()

	objectHandle := client.Bucket(gcp.bucket).Object(key)
	return objectHandle.Attrs(gcp.ctx)
}

func (gcp *storageImpl) FileExist(fp string) (bool, error) {
	_, err := gcp.GetAttr(fp)
	if err != nil {
		errIsObjNotExist := strings.Contains(err.Error(), "object doesn't exist")
		if errIsObjNotExist {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

const (
	_Member_AllUsers   = "allUsers"
	_Role_ObjectReader = iam.RoleName("roles/storage.legacyObjectReader")
)

type DownloadUrl struct {
	IsPublic    bool
	Url         string
	AccessToken *oauth2.Token
}

func (gcp *storageImpl) GetDownloadUrl(key string) (myurl *DownloadUrl, err error) {
	client, err := gcp.getClient()
	if err != nil {
		err = fmt.Errorf("storage.NewClient: %v", err)
		return
	}
	defer client.Close()

	bucketHandler := client.Bucket(gcp.bucket)

	policy, err := bucketHandler.IAM().Policy(gcp.ctx)
	if err != nil {
		return nil, err
	}
	members := policy.Members(_Role_ObjectReader)
	isPublic := false
	for _, m := range members {
		if m == _Member_AllUsers {
			isPublic = true
			break
		}
	}

	myurl = &DownloadUrl{
		IsPublic: isPublic,
	}

	if !isPublic {
		token, err := gcp.GetAccessToken()
		if err != nil {
			return nil, err
		}
		myurl.AccessToken = token
	}

	objectHandle := bucketHandler.Object(key)
	attrs, err := objectHandle.Attrs(gcp.ctx)
	if err != nil {
		return
	}

	u, err := url.Parse(attrs.MediaLink)
	if err != nil {
		return
	}
	rel, err := u.Parse(strAppend("/", gcp.bucket, "/", key))
	if err != nil {
		return
	}
	myurl.Url = rel.String()
	return
}

func (gcp *storageImpl) SignedURL(key string, contentType string, expDuration time.Duration) (url string, err error) {
	conf, err := google.JWTConfigFromJSON(gcp.jsonData)
	if err != nil {
		return
	}

	url, err = googstorage.SignedURL(gcp.bucket, key,
		&googstorage.SignedURLOptions{
			GoogleAccessID: conf.Email,
			Method:         "PUT",
			PrivateKey:     conf.PrivateKey,
			Expires:        time.Now().Add(expDuration),
			ContentType:    contentType,
		})
	return
}

func (gcp *storageImpl) GetAccessToken() (*oauth2.Token, error) {
	var c = struct {
		Email      string `json:"client_email"`
		PrivateKey string `json:"private_key"`
	}{}
	json.Unmarshal([]byte(gcp.jsonData), &c)
	config := &jwt.Config{
		Email:      c.Email,
		PrivateKey: []byte(c.PrivateKey),
		Scopes: []string{
			"https://www.googleapis.com/auth/devstorage.read_only",
		},
		TokenURL: google.JWTTokenURL,
	}
	token, err := config.TokenSource(context.Background()).Token()
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (gcp *storageImpl) Get(key string) ([]byte, error) {
	client, err := gcp.getClient()
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	rc, err := client.Bucket(gcp.bucket).Object(key).NewReader(gcp.ctx)
	if err != nil {
		return nil, fmt.Errorf("Object(%q).NewReader: %v", key, err)
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func (gcp *storageImpl) List(dir string) ([]string, error) {
	client, err := gcp.getClient()
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	result := []string{}
	it := client.Bucket(gcp.bucket).Objects(gcp.ctx, &googstorage.Query{
		Prefix: dir,
	})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Bucket(%q).Objects: %w", gcp.bucket, err)
		}
		if strings.HasSuffix(attrs.Name, "/") {
			continue
		}
		result = append(result, attrs.Name)
	}
	return result, nil
}

func filenameEncode(str string) string {
	w := md5.New()
	io.WriteString(w, str)
	return hex.EncodeToString(w.Sum(nil))
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

type GcpConfigMap interface {
	GetConfig(key string) *GcpConf
}

type gcpConfigMap map[string]*GcpConf

func (m *gcpConfigMap) GetConfig(key string) *GcpConf {
	return (*m)[key]
}

func LoadGcpConfigMap(file string) (GcpConfigMap, error) {
	result := make(gcpConfigMap)
	// read file
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	// Unmarshal yaml to result
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
