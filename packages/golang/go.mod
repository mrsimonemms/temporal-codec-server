module github.com/mrsimonemms/temporal-codec-server/packages/golang

go 1.24.3

replace github.com/mrsimonemms/temporal-codec-server/examples/golang => ../../examples/golang

require (
	github.com/MicahParks/keyfunc/v3 v3.4.0
	github.com/golang-jwt/jwt/v5 v5.2.3
	github.com/golang/snappy v1.0.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/mrsimonemms/temporal-codec-server/examples/golang v0.0.0-00010101000000-000000000000
	github.com/redis/go-redis/v9 v9.18.0
	github.com/stretchr/testify v1.10.0
	go.temporal.io/api v1.50.0
	go.temporal.io/sdk v1.35.0
	sigs.k8s.io/yaml v1.5.0
)

require (
	github.com/MicahParks/jwkset v0.9.6 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.1 // indirect
	github.com/nexus-rpc/sdk-go v0.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	google.golang.org/grpc v1.73.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
