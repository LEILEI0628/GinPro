package session

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func CreateSession(context *gin.Context, userId int64, options sessions.Options) error {
	session := sessions.Default(context)
	session.Set("userId", userId)
	session.Options(options)
	err := session.Save()
	return err
}
