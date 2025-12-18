package v1

const (
	CursorDelimiter = "_"
)

type chainIDMap map[int]string

// chainIDToChain 链 ID 到链名称的映射配置
// 用于将数字 ID 转换为人类可读的字符串标识
var chainIDToChain = chainIDMap{
	1:        "eth",
	10:       "optimism",
	11155111: "sepolia",
}
