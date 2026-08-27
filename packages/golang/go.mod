module github.com/mrsimonemms/temporal-codec-server/packages/golang

go 1.25.4

replace github.com/mrsimonemms/temporal-codec-server/examples/golang => ../../examples/golang

require (
	github.com/MicahParks/keyfunc/v3 v3.8.1
	github.com/aws/aws-sdk-go-v2 v1.45.0
	github.com/aws/aws-sdk-go-v2/config v1.33.0
	github.com/aws/aws-sdk-go-v2/credentials v1.20.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.109.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/golang/snappy v1.0.0
	github.com/google/uuid v1.6.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/redis/go-redis/v9 v9.22.0
	go.temporal.io/api v1.63.5
	go.temporal.io/sdk v1.48.0
	sigs.k8s.io/yaml v1.6.0
)

require (
	github.com/MicahParks/jwkset v0.11.3 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.20 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.19.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.5.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.11.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.14.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.20.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.7.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.35.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.40.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.47.0 // indirect
	github.com/aws/smithy-go v1.28.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.30.0 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/nexus-rpc/nexus-proto-annotations v0.1.0 // indirect
	github.com/nexus-rpc/sdk-go v0.7.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260825221802-da73d73af1c5 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260825221802-da73d73af1c5 // indirect
	google.golang.org/grpc v1.83.2 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
