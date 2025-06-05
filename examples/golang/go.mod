module golang

go 1.24.3

replace github.com/mrsimonemms/temporal-codec-server/pkg/temporal => ../../

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20250528174236-200df99c418a

require (
	github.com/golang/mock v1.7.0-rc.1
	github.com/mrsimonemms/temporal-codec-server/pkg/temporal v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.10.0
	go.temporal.io/api v1.49.1
	go.temporal.io/sdk v1.34.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.3 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mrsimonemms/temporal-codec-server v0.0.0-20250530100957-fd801a2f056a // indirect
	github.com/nexus-rpc/sdk-go v0.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/grpc v1.73.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
