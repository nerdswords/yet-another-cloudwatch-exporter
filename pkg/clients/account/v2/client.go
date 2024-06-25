package v2

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/account"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
)

type client struct {
	logger    logging.Logger
	stsClient *sts.Client
	iamClient *iam.Client
}

func NewClient(logger logging.Logger, stsClient *sts.Client, iamClient *iam.Client) account.Client {
	return &client{
		logger:    logger,
		stsClient: stsClient,
		iamClient: iamClient,
	}
}

func (c client) GetAccount(ctx context.Context) (string, error) {
	result, err := c.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	if result.Account == nil {
		return "", errors.New("aws sts GetCallerIdentityWithContext returned no account")
	}
	return *result.Account, nil
}

func (c client) GetAccountAlias(ctx context.Context) (string, error) {
	acctAliasOut, err := c.iamClient.ListAccountAliases(ctx, &iam.ListAccountAliasesInput{})
	if err != nil {
		return "", err
	}

	possibleAccountAlias := ""

	// Since a single account can only have one alias, and an authenticated SDK session corresponds to a single account,
	// the output can have at most one alias.
	// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListAccountAliases.html
	if len(acctAliasOut.AccountAliases) > 0 {
		possibleAccountAlias = acctAliasOut.AccountAliases[0]
	}

	return possibleAccountAlias, nil
}
