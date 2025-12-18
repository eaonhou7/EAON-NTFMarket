package types

// LoginReq 用户登录请求
type LoginReq struct {
	ChainID   int    `json:"chain_id"`  // 链 ID
	Message   string `json:"message"`   // 签名消息 (Nonce)
	Signature string `json:"signature"` // 签名结果
	Address   string `json:"address"`   // 用户地址
}

// UserLoginInfo 登录成功响应
type UserLoginInfo struct {
	Token     string `json:"token"`      // 鉴权 Token (JWT/Session)
	IsAllowed bool   `json:"is_allowed"` // 是否允许登录
}

type UserLoginResp struct {
	Result interface{} `json:"result"`
}

// UserLoginMsgResp 登录消息响应
type UserLoginMsgResp struct {
	Address string `json:"address"` // 用户地址
	Message string `json:"message"` // 生成的随机 Nonce 消息
}

// UserSignStatusResp 用户签名状态
type UserSignStatusResp struct {
	IsSigned bool `json:"is_signed"` // true: 已注册/签名过, false: 新用户
}
