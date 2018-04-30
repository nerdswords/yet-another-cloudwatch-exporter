package main

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"strconv"
)

func getAwsArn() *string {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	svc := sts.New(sess)

	input := &sts.GetCallerIdentityInput{}

	result, err := svc.GetCallerIdentity(input)
	if err != nil {
		panic(err)
	}

	return result.Account
}

func intToString(n *int64) *string {
	label := strconv.FormatInt(*n, 10)
	return &label
}
