// Command ota 用发行目录里的 SHA256SUMS.txt 和安装包写出带签名的更新频道 JSON。
//
// 发布流程里调用。它只生成并签名频道文件，不负责上传到 ModelScope。签名私钥的解析复用 internal/update 的 ParsePrivateKey。
package main
