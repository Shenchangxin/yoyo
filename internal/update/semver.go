package update

import (
	"strconv"
	"strings"
)

func canon(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	return v
}

type parsed struct {
	major, minor, patch int
	pre                 string
	ok                  bool
}

func parseSemver(v string) parsed {
	v = canon(v)
	if v == "" {
		return parsed{}
	}
	core, pre, _ := strings.Cut(v, "-")
	core, _, _ = strings.Cut(core, "+")
	parts := strings.Split(core, ".")
	if len(parts) < 1 {
		return parsed{}
	}
	nums := [3]int{}
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return parsed{}
		}
		nums[i] = n
	}
	return parsed{major: nums[0], minor: nums[1], patch: nums[2], pre: pre, ok: true}
}

// CompareVersions returns -1 if a<b, 0 if equal, 1 if a>b (semver, v optional).
func CompareVersions(a, b string) int {
	pa, pb := parseSemver(a), parseSemver(b)
	if !pa.ok && !pb.ok {
		return strings.Compare(canon(a), canon(b))
	}
	if !pa.ok {
		return -1
	}
	if !pb.ok {
		return 1
	}
	if pa.major != pb.major {
		return cmpInt(pa.major, pb.major)
	}
	if pa.minor != pb.minor {
		return cmpInt(pa.minor, pb.minor)
	}
	if pa.patch != pb.patch {
		return cmpInt(pa.patch, pb.patch)
	}
	if pa.pre == pb.pre {
		return 0
	}
	if pa.pre == "" {
		return 1
	}
	if pb.pre == "" {
		return -1
	}
	return strings.Compare(pa.pre, pb.pre)
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func IsNewer(remote, current string) bool {
	if strings.TrimSpace(remote) == "" {
		return false
	}
	return CompareVersions(remote, current) > 0
}
