package collectors

import (
	"sort"
	"strings"
)

// labelMapToString converts a map of labels to a sorted string representation for deduping
func labelMapToString(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(labels[k])
		sb.WriteString(";")
	}
	return sb.String()
}
