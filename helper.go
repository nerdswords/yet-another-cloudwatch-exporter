package main

import (
	"strconv"
)

func intToString(n *int64) *string {
	label := strconv.FormatInt(*n, 10)
	return &label
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}
