// Package security 提供企业级数据安全功能。
//
// 核心功能：
//   - AES-256-GCM 加密
//   - 字段级加密
//   - 数据脱敏（邮箱、手机号、身份证、银行卡）
//   - 密钥管理
//
// 使用示例：
//
//	// AES 加密
//	key := security.GenerateKey()
//	encryptor := security.NewAESEncryptor(key)
//	ciphertext, _ := encryptor.Encrypt([]byte("sensitive data"))
//	plaintext, _ := encryptor.Decrypt(ciphertext)
//
//	// 数据脱敏
//	emailMasker := security.NewEmailMasker()
//	masked := emailMasker.Mask("user@example.com") // -> u***@example.com
//
//	phoneMasker := security.NewPhoneMasker()
//	masked := phoneMasker.Mask("13812345678") // -> 138****5678
//
package security
