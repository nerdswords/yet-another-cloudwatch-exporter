package main

import (
	"strings"
)

func ConvertTagToLabel(label string) string {
	replacer := strings.NewReplacer(" ", "_", ",", "_", "\t", "_", ",", "_", "/", "_", "\\", "_", ".", "_", "-", "_")
	return replacer.Replace(label)
}
