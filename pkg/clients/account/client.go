package account

import "context"

type Client interface {
	GetAccount(ctx context.Context) (string, error)

	// GetAccountAlias returns the account alias if there's one set, otherwise an empty string.
	GetAccountAlias(ctx context.Context) (string, error)
}
