package utils

import (
	"errors"
	"time"

	"github.com/BinLe1988/multi-agent-chatter/configs"

	"github.com/golang-jwt/jwt/v4"
)

// 全局JWT密钥
var jwtSecret string
var jwtExpiration int

// 初始化JWT配置
func InitJWT(cfg *configs.Config) {
	jwtSecret = cfg.JWT.Secret
	jwtExpiration = cfg.JWT.ExpiresIn
}

// Claims JWT声明
type Claims struct {
	UserID uint `json:"user_id"`
	jwt.StandardClaims
}

// GenerateToken 生成JWT令牌
func GenerateToken(userID uint) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(time.Duration(jwtExpiration) * time.Hour)

	claims := Claims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer:    "multi-agent-chatter",
		},
	}

	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString([]byte(jwtSecret))

	return token, err
}

// ParseToken 解析JWT令牌
func ParseToken(token string) (*Claims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
