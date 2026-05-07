package common

import "strings"

func EnsureHTTPSPrefix(url string) string {
	if url != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	return url
}
