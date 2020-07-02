/**
 * @Author: lyszhang
 * @Email: zhangliang@link-logis.com
 * @Date: 2020/6/29 4:32 PM
 */

package logmgr

import (
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

func GoID() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, _ := strconv.Atoi(idField)
	return id
}

func GoPackage() string {
	type em struct{}
	return reflect.TypeOf(em{}).PkgPath()
}

func GoPodname() string {
	return os.Getenv("HOSTNAME")
}

func GoNamespace() string {
	return "tmp"
}
