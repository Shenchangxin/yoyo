// uv_pin.go 钉死托管 uv 的版本和每个平台归档的 sha256（取自 GitHub 发布页的 digest）。升级 uv 时只改这里。

package buddy

// uvVersion 是托管 uv 的版本。
const uvVersion = "0.13.7"

// uvSHA256 是各平台归档的 sha256，键是归档文件名。
var uvSHA256 = map[string]string{
	"uv-x86_64-pc-windows-msvc.zip":        "0cdc5b38fe5a8168ed6a25814f5ba43c21d313dd3187314fccb37b7cec46e021",
	"uv-aarch64-pc-windows-msvc.zip":       "505d9e65e7692bac3ddf98c66754aa26c5050c409c184af30a881a0a72cc85e3",
	"uv-x86_64-apple-darwin.tar.gz":        "20c47e4c1f630b8abaff8c291fa9b9aa300fef92c4fbff95adf198b55287cda7",
	"uv-aarch64-apple-darwin.tar.gz":       "81b811b57e4624df7ed3aac148e0359100d9962f3d9dce00aec533dc2e086731",
	"uv-x86_64-unknown-linux-gnu.tar.gz":   "c5221ff41b7892bfc223543883600125f6a47fe845093e0f5bdff4f359196d83",
	"uv-aarch64-unknown-linux-gnu.tar.gz":  "71279f8d3c67241e807b9d8db872d1bf541579fb9f08b7d1eb1682196d1e57dc",
	"uv-x86_64-unknown-linux-musl.tar.gz":  "287491da03f2a785c70fa75b13e1f849918c2f2b5415fe84876b8214eeb792f9",
	"uv-aarch64-unknown-linux-musl.tar.gz": "2d3ec076a0c4c0e8250bb6418c44080828b544f1a3be0b6e46adf9fb6923de45",
}
