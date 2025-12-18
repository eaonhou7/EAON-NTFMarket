package service

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/base"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/ProjectsTask/EasySwapBackend/src/api/middleware"
	"github.com/ProjectsTask/EasySwapBackend/src/service/svc"
	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

func getUserLoginMsgCacheKey(address string) string {
	return middleware.CR_LOGIN_MSG_KEY + ":" + strings.ToLower(address)
}

func getUserLoginTokenCacheKey(address string) string {
	return middleware.CR_LOGIN_KEY + ":" + strings.ToLower(address)
}

// UserLogin 用户登录接口
// 功能:
// 1. 验证用户签名 (Signature Verification) [注: 目前验证逻辑被注释，使用了 Mock 验证]
// 2. 验证 Nonce 有效性 (防止重放攻击)
// 3. 生成并缓存 JWT/Token (用于后续接口鉴权)
// 4. 如果用户不存在则自动注册 (Auto Register)
func UserLogin(ctx context.Context, svcCtx *svc.ServerCtx, req types.LoginReq) (*types.UserLoginInfo, error) {
	// 返回结果容器
	res := types.UserLoginInfo{}

	// 1. 签名验证 (TODO: 生产环境必须开启)
	//todo: add verify signature
	//ok := verifySignature(req.Message, req.Signature, req.PublicKey)
	//if !ok {
	//	return nil, errors.New("invalid signature")
	//}

	// 2. 验证 Nonce (防止重放攻击)
	// 2.1 从缓存中获取该地址对应的登录消息 UUID (Key: prefix + userAddr)
	cachedUUID, err := svcCtx.KvStore.Get(getUserLoginMsgCacheKey(req.Address))
	if cachedUUID == "" || err != nil {
		// 如果缓存中没有 Nonce, 说明可能已过期或从未申请过
		return nil, errcode.ErrTokenExpire
	}

	// 2.2 解析前端传递的消息以获取 Nonce
	// 预期消息格式: "Welcome to EasySwap!\nNonce:<uuid>"
	splits := strings.Split(req.Message, "Nonce:")
	if len(splits) != 2 {
		return nil, errcode.ErrTokenExpire
	}

	// 2.3 比对 Nonce 是否一致
	loginUUID := strings.Trim(splits[1], "\n")
	if loginUUID != cachedUUID {
		return nil, errcode.ErrTokenExpire
	}

	// 3. 查询或创建用户信息 (Auto Register)
	var user base.User
	db := svcCtx.DB.WithContext(ctx).Table(base.UserTableName()).
		Select("id,address,is_allowed").
		Where("address = ?", req.Address).
		Find(&user)
	if db.Error != nil {
		return nil, errors.Wrap(db.Error, "failed on get user info")
	}

	// 如果用户不存在则创建新用户
	if user.Id == 0 {
		now := time.Now().UnixMilli()
		user := &base.User{
			Address:    req.Address,
			IsAllowed:  false, // 默认未授权
			IsSigned:   true,  // 标记为已签名登录
			CreateTime: now,
			UpdateTime: now,
		}
		if err := svcCtx.DB.WithContext(ctx).Table(base.UserTableName()).
			Create(user).Error; err != nil {
			return nil, errors.Wrap(db.Error, "failed on create new user")
		}
	}

	// 4. 生成并缓存 User Token
	// tokenKey: login_token_key + userAddress
	tokenKey := getUserLoginTokenCacheKey(req.Address)

	// 使用 AES 加密生成 Token (TODO: 建议使用标准 JWT)
	userToken, err := AesEncryptOFB([]byte(tokenKey), []byte(middleware.CR_LOGIN_SALT))
	if err != nil {
		return nil, errors.Wrap(err, "failed on get user token")
	}

	// 缓存 Token (Value: UUID, TTL: 30天)
	if err := CacheUserToken(svcCtx, tokenKey, uuid.NewString()); err != nil {
		return nil, err
	}

	// 5. 设置返回结果
	res.Token = hex.EncodeToString(userToken)
	res.IsAllowed = user.IsAllowed

	return &res, err
}

// CacheUserToken 将用户 Token 写入 Redis 缓存
// Key: login_token_key:address
// Value: uuid (随机值, 目前似乎仅用于占位或简单验证)
// TTL: 30天
func CacheUserToken(svcCtx *svc.ServerCtx, tokenKey, token string) error {
	if err := svcCtx.KvStore.Setex(tokenKey, token, 30*24*60*60); err != nil {
		return err
	}

	return nil
}

// AesEncryptOFB 使用 AES-OFB 模式进行加密
// 参数:
// - data: 待加密数据
// - key: 加密密钥
func AesEncryptOFB(data []byte, key []byte) ([]byte, error) {
	// 对数据进行 PKCS7 填充，确保长度符合 AES 块大小要求
	data = PKCS7Padding(data, aes.BlockSize)
	// 创建 AES Cipher
	block, _ := aes.NewCipher([]byte(key))
	out := make([]byte, aes.BlockSize+len(data))
	// 随机生成 IV (初始化向量)
	iv := out[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// 使用 OFB 模式加密
	stream := cipher.NewOFB(block, iv)
	stream.XORKeyStream(out[aes.BlockSize:], data)
	return out, nil
}

// PKCS7Padding 补码
// PKCS7Padding 负责对 AES 加密数据块进行填充
// AES加密数据块分组长度必须为128bit(byte[16])，密钥长度可以是128bit(byte[16])、192bit(byte[24])、256bit(byte[32])中的任意一个。
func PKCS7Padding(ciphertext []byte, blocksize int) []byte {
	padding := blocksize - len(ciphertext)%blocksize
	// 填充 padding 个 byte(padding)
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func genLoginTemplate(nonce string) string {
	return fmt.Sprintf("Welcome to EasySwap!\nNonce:%s", nonce)
}

// GetUserLoginMsg 生成并返回用户的登录签名消息 (Nonce)
// 功能:
// 1. 生成一个随机的 UUID 作为 Nonce，防止重放攻击
// 2. 将 UUID 存入 Redis，设置过期时间 (72小时)
// 3. 返回包含 Nonce 的签名原文，供前端进行 Web3 签名
func GetUserLoginMsg(ctx context.Context, svcCtx *svc.ServerCtx, address string) (*types.UserLoginMsgResp, error) {
	uuid := uuid.NewString() // 生成唯一标识
	loginMsg := genLoginTemplate(uuid)
	// 将 UUID 存入 redis，有效期 72 小时
	// Key: login_message_prefix + userAddress
	if err := svcCtx.KvStore.Setex(getUserLoginMsgCacheKey(address), uuid, 72*60*60); err != nil {
		return nil, errors.Wrap(err, "failed on generate login msg")
	}

	return &types.UserLoginMsgResp{Address: address, Message: loginMsg}, nil
}

// GetSigStatusMsg 查询用户的签名状态
// 功能:
// 1. 查询用户是否已经完成了签名/注册流程
// 2. 只有完成签名的用户才能进行后续的敏感操作
func GetSigStatusMsg(ctx context.Context, svcCtx *svc.ServerCtx, userAddr string) (*types.UserSignStatusResp, error) {
	isSigned, err := svcCtx.Dao.GetUserSigStatus(ctx, userAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get user sign status")
	}

	return &types.UserSignStatusResp{IsSigned: isSigned}, nil
}
