package storage

import (
	"github.com/94peter/log"
	"github.com/94peter/microservice/cfg"
	"github.com/94peter/microservice/di"
	"github.com/pkg/errors"
)

type ModelDI interface {
	log.LoggerDI
}

type Config struct {
	ConfMapPath string `env:"GCP_CONF_MAP_PATH"`

	ModelDI

	ConfMap GcpConfigMap
	Log     log.Logger
}

func GetConfigFromEnv() (*Config, error) {
	var mycfg Config
	err := cfg.GetFromEnv(&mycfg)
	if err != nil {
		return nil, err
	}

	mycfg.ConfMap, err = LoadGcpConfigMap(mycfg.ConfMapPath)
	if err != nil {
		return nil, err
	}
	return &mycfg, nil
}

func (c *Config) Close() error {
	return nil
}
func (c *Config) Init(uuid string, di di.DI) error {
	mdi, ok := di.(ModelDI)
	if !ok {
		return errors.New("no ModelDI")
	}
	var err error

	c.ModelDI = mdi
	c.Log, err = mdi.NewLogger(di.GetService(), uuid)
	if err != nil {
		return err
	}
	return nil
}
func (c *Config) Copy() cfg.ModelCfg {
	cfg := *c
	return &cfg
}
