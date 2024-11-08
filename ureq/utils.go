package ureq

import (
	"net/url"
	"sort"
	"strings"
)

func encodeUrlValues(v url.Values, keys []string) string {
	if len(v) == 0 {
		return ""
	}
	var buf strings.Builder

	if len(keys) < 1 {
		keys = make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
	}

	for _, k := range keys {
		vs := v[k]
		keyEscaped := url.QueryEscape(k)
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
		}
	}
	return buf.String()
}
