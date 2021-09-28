package exporter

import (
	"os"
	"strconv"
)

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func GetEnv(key string, defaultValue interface{}) interface{} {
	val, ok := os.LookupEnv(key)
	switch key {
	case "debug":
		if ok {
			value, _ := strconv.ParseBool(val)
			return value
		} else {
			return defaultValue
		}
	case "addr":
		if ok {
			return val
		} else {
			return defaultValue
		}
	case "scrapingInterval":
		if ok {
			value, _ := strconv.Atoi(val)
			return value
		} else {
			return defaultValue
		}
	case "metricsPerQuery":
		if ok {
			value, _ := strconv.Atoi(val)
			return value
		} else {
			return defaultValue
		}
	default:
		return nil
	}
}
