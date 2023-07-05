package account

import "context"

type Client interface {
	GetAccount(ctx context.Context) (string, error)
}
