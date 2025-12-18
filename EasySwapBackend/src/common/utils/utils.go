package utils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/go-playground/validator/v10"
)

var (
	// validatorM 存储自定义的验证器函数映射
	// key: 验证规则名称 ("symbol", "address")
	// value: 验证函数实现
	validatorM map[string]validator.Func
	// patternM 存储正则表达式模式映射
	// key: 验证规则名称
	// value: 正则表达式字符串
	patternM map[string]string
)

// init 初始化验证器和正则模式
func init() {
	// 初始化验证函数映射
	validatorM = map[string]validator.Func{
		"symbol":  rightSymbol,     // 验证代币符号长度
		"address": regexpValidator, // 使用正则验证地址格式
	}
	// 初始化正则模式映射
	patternM = map[string]string{
		// 以太坊地址正则: 0x开头,后接40位16进制字符
		"address": `^0x[a-fA-F0-9]{40}$`,
	}
}

var (
	// rightSymbol 验证代币符号(Symbol)是否合法
	// 规则: 必须是字符串且长度小于 10
	rightSymbol validator.Func = func(fl validator.FieldLevel) bool {
		// 尝试将字段值断言为 string 类型
		symbol, ok := fl.Field().Interface().(string)
		if ok {
			// 如果是字符串,检查长度是否小于 10
			return len(symbol) < 10
		}
		// 如果不是字符串类型,验证失败
		return false
	}

	// regexpValidator 通用正则验证器
	// 功能: 根据 tag 中指定的模式名称(如 "address")查找对应的正则表达式并进行匹配
	regexpValidator validator.Func = func(fl validator.FieldLevel) bool {
		// 获取字段值字符串
		key, _ := fl.Field().Interface().(string)
		// 从 patternM 中查找对应的正则表达式
		pattern, ok := patternM[fl.GetTag()]
		if ok {
			// 如果找到了正则模式,执行匹配
			match, _ := regexp.MatchString(pattern, key)
			return match
		}
		// 如果没找到对应的正则模式,验证失败
		return false
	}
)

// ToValidateAddress 将以太坊地址转换为校验和格式 (Checksum Address)
// 功能: 遵循 EIP-55 规范,将地址转换为混合大小写的校验和格式
// 原理: 对地址的小写形式进行 Keccak-256 哈希,根据哈希值的每一位决定对应字符的大小写
func ToValidateAddress(address string) string {
	addrLowerStr := strings.ToLower(address)
	if strings.HasPrefix(addrLowerStr, "0x") {
		addrLowerStr = addrLowerStr[2:]
		address = address[2:]
	}
	var binaryStr string
	addrBytes := []byte(addrLowerStr)

	hash256 := common.Keccak256Hash([]byte(addrLowerStr)) //注意，这里是直接对字符串转换成byte切片然后哈希

	// 遍历地址的每个字符
	for i, e := range addrLowerStr {
		// 如果是数字(0-9),则不需要转换大小写,直接跳过
		if e >= '0' && e <= '9' {
			continue
		} else {
			// 将哈希值的第 i/2 个字节转换为二进制字符串 (8位,不足补0)
			// 注意: 一个字节对应地址中的两个十六进制字符
			binaryStr = fmt.Sprintf("%08b", hash256[i/2])
			// 检查二进制字符串的特定位
			// i%2==0 时检查高4位(索引0), i%2==1 时检查低4位(索引4)
			// 如果该位是 '1',则将对应字符转换为大写
			if binaryStr[4*(i%2)] == '1' {
				// ASCII码小写转大写: 减去 32
				addrBytes[i] -= 32
			}
		}
	}

	return "0x" + string(addrBytes)
}
