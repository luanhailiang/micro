package matedata

import (
	"os"

	"github.com/golang-jwt/jwt"
	"github.com/luanhailiang/micro/proto/rpcmsg"
	"github.com/sirupsen/logrus"
)

var jwt_sign []byte = []byte("5656")

func init() {
	key := os.Getenv("JWT_TOKEN_KEY")
	if key != "" {
		jwt_sign = []byte(key)
	}
}

func JwtToken(msg *rpcmsg.MateMessage) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"space": msg.Space,
		"index": msg.Index,
		"admin": msg.Admin,
		"usrid": msg.Usrid,
		"sdkid": msg.Sdkid,
		"suuid": msg.Suuid,
		"count": msg.Count,
		"clnip": msg.Clnip,
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(jwt_sign)
	if err != nil {
		logrus.Errorf("token:%s", err.Error())
		return "", err
	}

	return tokenString, nil
}

func JwtMate(tokenString string) (*rpcmsg.MateMessage, error) {
	type MyCustomClaims struct {
		Space string `json:"space"`
		Index string `json:"index"`
		Admin string `json:"admin"`
		Usrid string `json:"usrid"`
		Sdkid string `json:"sdkid"`
		Suuid string `json:"suuid"`
		Count uint64 `json:"count"`
		Clnip string `json:"clnip"`
		jwt.StandardClaims
	}
	//生成token对象
	token, err := jwt.ParseWithClaims(tokenString, &MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_TOKEN_KEY")), nil
	})
	if err != nil {
		return nil, err
	}
	//解析token对象
	claims, ok := token.Claims.(*MyCustomClaims)
	if !ok || !token.Valid {
		return nil, err
	}
	//绑定角色信息
	mate := &rpcmsg.MateMessage{
		Space: claims.Space,
		Index: claims.Index,
		Admin: claims.Admin,
		Usrid: claims.Usrid,
		Sdkid: claims.Sdkid,
		Suuid: claims.Suuid,
		Count: claims.Count,
		Clnip: claims.Clnip,
	}
	return mate, nil
}
