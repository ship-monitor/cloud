package specs

//go:generate go tool ogen -target ../internal/v2/gen/api -clean -config ./.ogen.yml ./openapi.yml

//go:generate go tool go-asyncapi code -M github.com/ship-monitor/cloud/pkg/connector -t ../pkg/connector ./asyncapi.yml
