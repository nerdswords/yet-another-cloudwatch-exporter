package main

import (
	"strings"
)

func ConvertTagToLabel(label string) string {
	replacer := strings.NewReplacer(" ", "_", ",", "_", "\t", "_", ",", "_", "/", "_", "\\", "_", ".", "_", "-", "_")
	saveLabel := replacer.Replace(label)
	return "tag_" + saveLabel
}
