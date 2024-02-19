# Storage Service

專案用來管理檔案存取服務，支援google storage

### Develop config file sample
```yaml
log:
    fluentd:
      host: fluentd
      port: 24224

```


## Environment Variable

```env
# name for service as log prefix
SERVICE=storage

# assign grpc service port
GRPC_PORT=7080

# set config file root path
CONF_PATH=./config/

# log level
LOG_LEVEL=debug|info|warn|error|fatal

# log message send to target
LOG_TARGET=os|fluent

# gcp credential files mapping
GCP_CONF_MAP_PATH=/etc/gcp_config_map.yml
```

## Gcp Config Map
```yaml
default:
  credentailsFile: "/etc/gcp_credentials_files/muulin-universal.json"
  bucket: "pub.storage.muulin-tech.com"
```