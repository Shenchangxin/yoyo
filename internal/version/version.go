package version

// Version is the user-visible product version.
// Release builds override it with:
//
//	-ldflags "-X github.com/Shenchangxin/yoyo/internal/version.Version=x.y.z"
var Version = "0.1.0"
