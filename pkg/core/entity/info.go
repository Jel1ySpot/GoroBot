package entity

import "strings"

type Info struct {
	Protocol string
	Args     []string
}

func ParseInfo(str string) (Info, bool) {
	s := strings.Split(str, ":")
	if len(s) != 2 {
		return Info{}, false
	}
	return Info{Protocol: s[0], Args: strings.Split(s[1], "&")}, true
}
