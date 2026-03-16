package utils

/*
JWT (JSON Web Token) 是一种带有数字签名的加密字符串。后端把它发给前端，前端下次请求带上它，后端只要验证签名没错，就能直接从中解析出用户 ID，别人绝对无法篡改。
*/
import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

// 密钥
var jwtSecret = []byte("Xiaohongshu_Simulator_Secret_Key_888")

// MyClaims 自定义声明结构体
type MyClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 生成jwt Token
func GenerateToken(userID uint, username string) (string, error) {
	claims := MyClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "Xiaohongshu_App",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken 解析并校验 JWT Token
func ParseToken(tokenString string) (*MyClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*MyClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
