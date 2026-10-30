package im

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

func sha1Join(parts ...string) string {
	sort.Strings(parts)
	sum := sha1.Sum([]byte(strings.Join(parts, "")))
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(secret string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write(body)
	return base64.StdEncoding.EncodeToString(m.Sum(nil))
}

func hmacSHA256Hex(secret string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

func slackSig(secret, ts string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write([]byte("v0:" + ts + ":"))
	_, _ = m.Write(body)
	return "v0=" + hex.EncodeToString(m.Sum(nil))
}

// decryptWeCom 按企业微信公布的 AES-CBC 格式解开密文：16 字节随机、4 字节长度、正文、接收方 id。
// 填充按 32 字节计，最后一字节可以大于 AES 的 16 字节块。
func decryptWeCom(encodingAESKey, encrypted string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return "", err
	}
	if len(key) != 32 {
		return "", fmt.Errorf("encoding aes key must be 32 bytes")
	}
	raw, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 || len(raw)%aes.BlockSize != 0 {
		return "", fmt.Errorf("bad ciphertext")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	plain := make([]byte, len(raw))
	cipher.NewCBCDecrypter(block, key[:aes.BlockSize]).CryptBlocks(plain, raw)
	plain, err = unpad32(plain)
	if err != nil {
		return "", err
	}
	if len(plain) < 20 {
		return "", fmt.Errorf("short plain")
	}
	n := int(binary.BigEndian.Uint32(plain[16:20]))
	if n < 0 || 20+n > len(plain) {
		return "", fmt.Errorf("bad message length")
	}
	return string(plain[20 : 20+n]), nil
}

func unpad32(plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return nil, fmt.Errorf("empty plain")
	}
	pad := int(plain[len(plain)-1])
	if pad <= 0 || pad > 32 || pad > len(plain) {
		return nil, fmt.Errorf("bad padding")
	}
	for _, b := range plain[len(plain)-pad:] {
		if int(b) != pad {
			return nil, fmt.Errorf("bad padding")
		}
	}
	return plain[:len(plain)-pad], nil
}

// decryptFeishu 解开飞书回调。密钥先做 SHA-256，再按 AES-256-CBC 去掉 PKCS7。
func decryptFeishu(key, encrypted string) (string, error) {
	sum := sha256.Sum256([]byte(strings.TrimSpace(key)))
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encrypted))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return "", err
	}
	if len(raw) > aes.BlockSize && (len(raw)-aes.BlockSize)%aes.BlockSize == 0 {
		if plain, err := cbcUnpad(block, raw[:aes.BlockSize], raw[aes.BlockSize:]); err == nil && jsonObject(plain) {
			return string(plain), nil
		}
	}
	if len(raw) > 0 && len(raw)%aes.BlockSize == 0 {
		if plain, err := cbcUnpad(block, sum[:aes.BlockSize], raw); err == nil && jsonObject(plain) {
			return string(plain), nil
		}
	}
	return "", fmt.Errorf("feishu decrypt failed")
}

func cbcUnpad(block cipher.Block, iv, raw []byte) ([]byte, error) {
	if len(iv) != aes.BlockSize || len(raw) == 0 || len(raw)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("bad ciphertext")
	}
	plain := make([]byte, len(raw))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, raw)
	if len(plain) == 0 {
		return nil, fmt.Errorf("empty plain")
	}
	pad := int(plain[len(plain)-1])
	if pad <= 0 || pad > aes.BlockSize || pad > len(plain) {
		return nil, fmt.Errorf("bad padding")
	}
	for _, b := range plain[len(plain)-pad:] {
		if int(b) != pad {
			return nil, fmt.Errorf("bad padding")
		}
	}
	return plain[:len(plain)-pad], nil
}

func jsonObject(b []byte) bool {
	b = bytes.TrimSpace(b)
	return len(b) > 0 && b[0] == '{'
}
