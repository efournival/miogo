package main

import "strings"

func formatD(res string) string {
	res = strings.TrimRight(strings.TrimSpace(res), "/")

	if len(res) == 0 {
		return "/"
	}

	return res
}

func parentD(res string) string {
	return formatD(res[:strings.LastIndex(res, "/")+1])
}

func formatF(res string) (dir string, file string) {
	res = formatD(res)
	pos := strings.LastIndex(res, "/")

	file = strings.TrimSpace(res[pos+1:])
	dir = formatD(res[:pos])

	return
}

func jsonkv(key, value string) string {
	return `{"` + key + `": "` + value + `"}`
}
