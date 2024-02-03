package helper

import (
	"fmt"
	"strings"
)

func AddQuotation(str string) string {
	return fmt.Sprintf("%q", str)
}

func MyStringIf(b bool, s1, s2 string) string {
	if b {
		return s1
	} else {
		return s2
	}
}

func JoinKeys(m map[string]bool, delimiter string) string {
	var keys []string
	for key := range m {
		keys = append(keys, key)
	}
	return strings.Join(keys, delimiter)
}

func SliceContainsTarget(slice []string, target string) bool {
	for _, value := range slice {
		if value == target {
			return true
		}
	}
	return false
}
