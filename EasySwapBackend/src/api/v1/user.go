package v1

import (
	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/kit/validator"
	"github.com/ProjectsTask/EasySwapBase/xhttp"
	"github.com/gin-gonic/gin"

	"github.com/ProjectsTask/EasySwapBackend/src/service/svc"
	"github.com/ProjectsTask/EasySwapBackend/src/service/v1"
	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

// UserLoginHandler 处理用户登录请求
// 功能:
// 1. 接收前端提交的签名信息和 Nonce
// 2. 验证签名合法性 (EIP-191/712)
// 3. 验证通过后颁发 JWT 或 Session Token
// UserLoginHandler 处理用户登录请求
// 功能:
// 1. 接收前端提交的签名信息和 Nonce
// 2. 验证签名合法性 (EIP-191/712)
// 3. 验证通过后颁发 JWT 或 Session Token
func UserLoginHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := types.LoginReq{}
		// 1. 绑定并解析 JSON 请求体
		if err := c.BindJSON(&req); err != nil {
			xhttp.Error(c, err)
			return
		}

		// 2. 参数基本校验 (Validator)
		if err := validator.Verify(&req); err != nil {
			xhttp.Error(c, errcode.NewCustomErr(err.Error()))
			return
		}

		// 3. 调用 Service 执行登录逻辑 (验证签名、生成Token)
		res, err := service.UserLogin(c.Request.Context(), svcCtx, req)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr(err.Error()))
			return
		}

		// 4. 返回登录凭证
		xhttp.OkJson(c, types.UserLoginResp{
			Result: res,
		})
	}
}

// GetLoginMessageHandler 获取登录签名消息 (Nonce)
// 功能:
// 1. 生成唯一的随机字符串 (Nonce)
// 2. 缓存 Nonce 到 Redis，关联用户地址
// 3. 返回 Nonce 给前端供用户签名
func GetLoginMessageHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取用户地址参数
		address := c.Params.ByName("address")
		if address == "" {
			xhttp.Error(c, errcode.NewCustomErr("user addr is null"))
			return
		}

		// 2. 调用 Service 生成登录消息
		res, err := service.GetUserLoginMsg(c.Request.Context(), svcCtx, address)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr(err.Error()))
			return
		}

		// 3. 返回消息对象
		xhttp.OkJson(c, res)
	}
}

// GetSigStatusHandler 查询用户签名状态
// 功能: 检查指定用户地址是否已经完成过注册或签名验证流程
func GetSigStatusHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		userAddr := c.Params.ByName("address")
		if userAddr == "" {
			xhttp.Error(c, errcode.NewCustomErr("user addr is null"))
			return
		}

		res, err := service.GetSigStatusMsg(c.Request.Context(), svcCtx, userAddr)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr(err.Error()))
			return
		}

		xhttp.OkJson(c, res)
	}
}
