module github.com/TrueCloudLab/frostfs-s3-gw

go 1.18

require (
	github.com/TrueCloudLab/frostfs-api-go/v2 v2.0.0-20221212144048-1351b6656d68
	github.com/TrueCloudLab/frostfs-sdk-go v0.0.0-20230130120602-cf64ddfb143c
	github.com/aws/aws-sdk-go v1.44.6
	github.com/bluele/gcache v0.0.2
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/minio/sio v0.3.0
	github.com/nats-io/nats.go v1.13.1-0.20220121202836-972a071d373d
	github.com/nspcc-dev/neo-go v0.101.0
	github.com/panjf2000/ants/v2 v2.5.0
	github.com/prometheus/client_golang v1.13.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.15.0
	github.com/stretchr/testify v1.8.1
	github.com/urfave/cli/v2 v2.3.0
	go.uber.org/zap v1.24.0
	golang.org/x/crypto v0.4.0
	google.golang.org/grpc v1.52.0
	google.golang.org/protobuf v1.28.1
)

replace (
	github.com/TrueCloudLab/frostfs-api-go/v2 v2.0.0-20221212144048-1351b6656d68 => github.com/KirillovDenis/frostfs-api-go/v2 v2.11.2-0.20230221082308-ac00938fa447
	github.com/TrueCloudLab/frostfs-sdk-go v0.0.0-20230130120602-cf64ddfb143c => github.com/KirillovDenis/frostfs-sdk-go v0.0.0-20230221122223-9424a67fb108
)

require (
	github.com/TrueCloudLab/frostfs-contract v0.0.0-20221213081248-6c805c1b4e42 // indirect
	github.com/TrueCloudLab/frostfs-crypto v0.5.0
	github.com/TrueCloudLab/hrw v1.1.0 // indirect
	github.com/TrueCloudLab/rfc6979 v0.3.0 // indirect
	github.com/TrueCloudLab/tzhash v1.7.0 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20221202181307-76fa05c21b12 // indirect
	//github.com/aws/aws-sdk-go-v2 v1.16.7 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/golang-lru v0.6.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/nats-io/nats-server/v2 v2.7.1 // indirect
	github.com/nats-io/nkeys v0.3.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nspcc-dev/go-ordered-json v0.0.0-20220111165707-25110be27d22 // indirect
	github.com/nspcc-dev/neo-go/pkg/interop v0.0.0-20221202075445-cb5c18dc73eb // indirect
	github.com/nspcc-dev/rfc6979 v0.2.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.6 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.9.3 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	github.com/urfave/cli v1.22.5 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/exp v0.0.0-20221227203929-1b447090c38c // indirect
	golang.org/x/net v0.4.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.3.0 // indirect
	golang.org/x/term v0.3.0 // indirect
	golang.org/x/text v0.5.0 // indirect
	google.golang.org/genproto v0.0.0-20221227171554-f9683d7f8bef // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
