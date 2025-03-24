package jwtx

import "github.com/golang-jwt/jwt/v5"

func CreateJWT(verificationKey []byte, userClaims UserClaims) (string, error) {
	// 创建JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, userClaims)
	return token.SignedString(verificationKey)
}
