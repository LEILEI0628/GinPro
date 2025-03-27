package session

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func CreateSession(ctx *gin.Context, userId int64, options sessions.Options) error {
	session := sessions.Default(ctx)
	session.Set("userId", userId)
	session.Options(options)
	err := session.Save()
	return err
}
