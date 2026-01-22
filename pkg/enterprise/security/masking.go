package security

import (
	"regexp"
	"strings"
)

// Masker 脱敏器接口
type Masker interface {
	// Mask 脱敏数据
	//
	// 参数：
	//   - value: 原始值
	//
	// 返回：
	//   - string: 脱敏后的值
	Mask(value string) string
}

// EmailMasker 邮箱脱敏器
//
// 保留第一个字符和域名，其余用 *** 替代。
//
// 示例：user@example.com -> u***@example.com
type EmailMasker struct{}

// NewEmailMasker 创建邮箱脱敏器
func NewEmailMasker() *EmailMasker {
	return &EmailMasker{}
}

// Mask 脱敏邮箱
func (m *EmailMasker) Mask(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email // 无效邮箱，返回原值
	}
	
	username := parts[0]
	domain := parts[1]
	
	if len(username) == 0 {
		return email
	}
	
	// 保留第一个字符
	if len(username) == 1 {
		return username + "***@" + domain
	}
	
	return string(username[0]) + "***@" + domain
}

// PhoneMasker 手机号脱敏器
//
// 保留前3位和后4位，中间用 **** 替代。
//
// 示例：13812345678 -> 138****5678
type PhoneMasker struct{}

// NewPhoneMasker 创建手机号脱敏器
func NewPhoneMasker() *PhoneMasker {
	return &PhoneMasker{}
}

// Mask 脱敏手机号
func (m *PhoneMasker) Mask(phone string) string {
	// 去除非数字字符
	phone = regexp.MustCompile(`[^\d]`).ReplaceAllString(phone, "")
	
	if len(phone) < 11 {
		return phone // 长度不足，返回原值
	}
	
	// 中国手机号：138****5678
	return phone[:3] + "****" + phone[7:]
}

// IDCardMasker 身份证号脱敏器
//
// 保留前6位和后4位，中间用 **** 替代。
//
// 示例：110101199001011234 -> 110101********1234
type IDCardMasker struct{}

// NewIDCardMasker 创建身份证号脱敏器
func NewIDCardMasker() *IDCardMasker {
	return &IDCardMasker{}
}

// Mask 脱敏身份证号
func (m *IDCardMasker) Mask(idCard string) string {
	if len(idCard) != 15 && len(idCard) != 18 {
		return idCard // 长度不符，返回原值
	}
	
	if len(idCard) == 15 {
		// 15 位身份证：110101*****1234
		return idCard[:6] + "*****" + idCard[11:]
	}
	
	// 18 位身份证：110101********1234
	return idCard[:6] + "********" + idCard[14:]
}

// BankCardMasker 银行卡号脱敏器
//
// 保留前4位和后4位，中间用 **** 替代。
//
// 示例：6222021234567890123 -> 6222********0123
type BankCardMasker struct{}

// NewBankCardMasker 创建银行卡号脱敏器
func NewBankCardMasker() *BankCardMasker {
	return &BankCardMasker{}
}

// Mask 脱敏银行卡号
func (m *BankCardMasker) Mask(cardNo string) string {
	// 去除非数字字符
	cardNo = regexp.MustCompile(`[^\d]`).ReplaceAllString(cardNo, "")
	
	if len(cardNo) < 8 {
		return cardNo // 长度不足，返回原值
	}
	
	// 银行卡号：6222********0123
	return cardNo[:4] + "********" + cardNo[len(cardNo)-4:]
}

// NameMasker 姓名脱敏器
//
// 保留姓氏，其余用 * 替代。
//
// 示例：张三 -> 张*，李四四 -> 李**
type NameMasker struct{}

// NewNameMasker 创建姓名脱敏器
func NewNameMasker() *NameMasker {
	return &NameMasker{}
}

// Mask 脱敏姓名
func (m *NameMasker) Mask(name string) string {
	runes := []rune(name)
	if len(runes) == 0 {
		return name
	}
	
	if len(runes) == 1 {
		return name // 单字，返回原值
	}
	
	// 保留第一个字符，其余用 * 替代
	return string(runes[0]) + strings.Repeat("*", len(runes)-1)
}

// AddressMasker 地址脱敏器
//
// 保留省市，详细地址用 **** 替代。
//
// 示例：北京市朝阳区建国路1号 -> 北京市朝阳区****
type AddressMasker struct{}

// NewAddressMasker 创建地址脱敏器
func NewAddressMasker() *AddressMasker {
	return &AddressMasker{}
}

// Mask 脱敏地址
func (m *AddressMasker) Mask(address string) string {
	// 查找"区"、"县"、"市"的位置
	markers := []string{"区", "县", "旗"}
	
	for _, marker := range markers {
		index := strings.Index(address, marker)
		if index > 0 {
			// 保留到"区/县"，后续用 **** 替代
			return address[:index+len(marker)] + "****"
		}
	}
	
	// 未找到标记，保留前8个字符
	runes := []rune(address)
	if len(runes) <= 8 {
		return address
	}
	
	return string(runes[:8]) + "****"
}

// FieldMasker 字段脱敏器
//
// 用于对文档或对象中的特定字段进行脱敏。
type FieldMasker struct {
	maskers map[string]Masker
}

// NewFieldMasker 创建字段脱敏器
//
// 参数：
//   - maskers: 字段名 -> 脱敏器映射
//
// 返回：
//   - *FieldMasker: 字段脱敏器
func NewFieldMasker(maskers map[string]Masker) *FieldMasker {
	return &FieldMasker{
		maskers: maskers,
	}
}

// MaskFields 脱敏字段
//
// 参数：
//   - data: 数据（map[string]any）
//
// 返回：
//   - error: 错误
func (f *FieldMasker) MaskFields(data map[string]any) error {
	for field, masker := range f.maskers {
		value, ok := data[field]
		if !ok {
			continue
		}
		
		// 转换为字符串
		strValue, ok := value.(string)
		if !ok {
			continue
		}
		
		// 脱敏
		masked := masker.Mask(strValue)
		
		// 更新字段
		data[field] = masked
	}
	
	return nil
}
