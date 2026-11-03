package poi

import (
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func productName() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	name, _, err := k.GetStringValue("ProductName")
	if err != nil || strings.TrimSpace(name) == "" {
		return ""
	}
	build, _, _ := k.GetStringValue("CurrentBuild")
	n, _ := strconv.Atoi(build)
	if n >= 22000 && strings.HasPrefix(name, "Windows 10") {
		name = "Windows 11" + strings.TrimPrefix(name, "Windows 10")
	}
	return name
}
