package permission

import "github.com/authzed/authzed-go/v1"

type Service struct {
	spicedb *authzed.Client
}

func New(client *authzed.Client) *Service {
	return &Service{
		spicedb: client,
	}
}
