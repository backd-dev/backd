module github.com/backd-dev/backd

go 1.26.1

require (
	github.com/aws/aws-sdk-go-v2 v1.41.4 // reason: S3-compatible file storage for internal/storage package
	github.com/aws/aws-sdk-go-v2/service/s3 v1.97.1 // reason: S3 service client for file operations
	github.com/google/cel-go v0.27.0
	github.com/jackc/pgx/v5 v5.8.0
	github.com/rs/xid v1.6.0
	golang.org/x/crypto v0.49.0 // reason: HKDF key derivation and AES-256-GCM encryption for secrets management
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aws/aws-sdk-go-v2/credentials v1.19.12
	github.com/stretchr/testify v1.11.1
)

require (
	cel.dev/expr v0.25.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.20 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.20 // indirect
	github.com/aws/smithy-go v1.24.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.6.1 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	golang.org/x/exp v0.0.0-20240823005443-9b4947da3948 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240826202546-f6391c0de4c7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240826202546-f6391c0de4c7 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
