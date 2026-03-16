package middleware

import (
	"Xiaohongshu_Simulator/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

// AuthRequired 强制登录的中间件
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从cookie中获取token
		tokenString, err := c.Cookie("token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或已过期"})
			c.Abort() //请求拦截
			return
		}
		// 解析并验证token
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的身份凭证"})
			c.Abort()
			return
		}
		// 验证通过，讲用户ID存入本次请求中
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)

		c.Next() //放行
	}
}
