package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidKey 无效的密钥
	ErrInvalidKey = errors.New("security: invalid key")
	
	// ErrInvalidCiphertext 无效的密文
	ErrInvalidCiphertext = errors.New("security: invalid ciphertext")
	
	// ErrEncryptionFailed 加密失败
	ErrEncryptionFailed = errors.New("security: encryption failed")
	
	// ErrDecryptionFailed 解密失败
	ErrDecryptionFailed = errors.New("security: decryption failed")
)

// Encryptor 加密器接口
type Encryptor interface {
	// Encrypt 加密数据
	//
	// 参数：
	//   - plaintext: 明文
	//
	// 返回：
	//   - []byte: 密文
	//   - error: 错误
	Encrypt(plaintext []byte) ([]byte, error)
	
	// Decrypt 解密数据
	//
	// 参数：
	//   - ciphertext: 密文
	//
	// 返回：
	//   - []byte: 明文
	//   - error: 错误
	Decrypt(ciphertext []byte) ([]byte, error)
	
	// EncryptString 加密字符串（返回 Base64）
	//
	// 参数：
	//   - plaintext: 明文字符串
	//
	// 返回：
	//   - string: Base64 编码的密文
	//   - error: 错误
	EncryptString(plaintext string) (string, error)
	
	// DecryptString 解密字符串（从 Base64）
	//
	// 参数：
	//   - ciphertext: Base64 编码的密文
	//
	// 返回：
	//   - string: 明文字符串
	//   - error: 错误
	DecryptString(ciphertext string) (string, error)
}

// AESEncryptor AES-256-GCM 加密器
//
// 使用 AES-256-GCM（Galois/Counter Mode）提供认证加密。
type AESEncryptor struct {
	key   []byte
	gcm   cipher.AEAD
}

// NewAESEncryptor 创建 AES 加密器
//
// 参数：
//   - key: 32 字节密钥（AES-256）
//
// 返回：
//   - *AESEncryptor: 加密器
//   - error: 错误
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: key must be 32 bytes for AES-256", ErrInvalidKey)
	}
	
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("security: failed to create GCM: %w", err)
	}
	
	return &AESEncryptor{
		key: key,
		gcm: gcm,
	}, nil
}

// Encrypt 加密数据
func (e *AESEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	// 生成随机 nonce
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: failed to generate nonce: %v", ErrEncryptionFailed, err)
	}
	
	// 加密（nonce + ciphertext + tag）
	ciphertext := e.gcm.Seal(nonce, nonce, plaintext, nil)
	
	return ciphertext, nil
}

// Decrypt 解密数据
func (e *AESEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrInvalidCiphertext)
	}
	
	// 提取 nonce 和密文
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	
	// 解密
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	
	return plaintext, nil
}

// EncryptString 加密字符串（返回 Base64）
func (e *AESEncryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString 解密字符串（从 Base64）
func (e *AESEncryptor) DecryptString(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: invalid base64: %v", ErrInvalidCiphertext, err)
	}
	
	plaintext, err := e.Decrypt(data)
	if err != nil {
		return "", err
	}
	
	return string(plaintext), nil
}

// GenerateKey 生成随机 AES-256 密钥
//
// 返回：
//   - []byte: 32 字节随机密钥
//   - error: 错误
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("security: failed to generate key: %w", err)
	}
	return key, nil
}

// FieldEncryptor 字段加密器
//
// 用于加密文档或对象中的特定字段。
type FieldEncryptor struct {
	encryptor Encryptor
	fields    []string
}

// NewFieldEncryptor 创建字段加密器
//
// 参数：
//   - encryptor: 加密器
//   - fields: 需要加密的字段列表
//
// 返回：
//   - *FieldEncryptor: 字段加密器
func NewFieldEncryptor(encryptor Encryptor, fields []string) *FieldEncryptor {
	return &FieldEncryptor{
		encryptor: encryptor,
		fields:    fields,
	}
}

// EncryptFields 加密字段
//
// 参数：
//   - data: 数据（map[string]any）
//
// 返回：
//   - error: 错误
func (f *FieldEncryptor) EncryptFields(data map[string]any) error {
	for _, field := range f.fields {
		value, ok := data[field]
		if !ok {
			continue
		}
		
		// 转换为字符串
		strValue := fmt.Sprint(value)
		
		// 加密
		encrypted, err := f.encryptor.EncryptString(strValue)
		if err != nil {
			return fmt.Errorf("security: failed to encrypt field %s: %w", field, err)
		}
		
		// 更新字段
		data[field] = encrypted
	}
	
	return nil
}

// DecryptFields 解密字段
//
// 参数：
//   - data: 数据（map[string]any）
//
// 返回：
//   - error: 错误
func (f *FieldEncryptor) DecryptFields(data map[string]any) error {
	for _, field := range f.fields {
		value, ok := data[field]
		if !ok {
			continue
		}
		
		// 转换为字符串
		strValue, ok := value.(string)
		if !ok {
			continue
		}
		
		// 解密
		decrypted, err := f.encryptor.DecryptString(strValue)
		if err != nil {
			return fmt.Errorf("security: failed to decrypt field %s: %w", field, err)
		}
		
		// 更新字段
		data[field] = decrypted
	}
	
	return nil
}
