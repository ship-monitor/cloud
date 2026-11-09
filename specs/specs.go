package specs

//go:generate go tool ogen -target ../internal/v2/gen/api -clean -config ./.ogen.yml ./openapi.yml

//go:generate go tool asyncapi-codegen -i ./asyncapi.yml -o ../pkg/connector/asyncapi/asyncapi.gen.go
