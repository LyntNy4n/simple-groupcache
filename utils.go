package simplegroupcache

import (
	"strings"
)

// 判断是否满足 x.x.x.x:port 的格式
func validPeerAddr(addr string) bool {
	token1 := strings.Split(addr, ":")
	if len(token1) != 2 {
		return false
	}
	token2 := strings.Split(token1[0], ".")
	if token1[0] != "localhost" && len(token2) != 4 {
		return false
	}
	return true
}
