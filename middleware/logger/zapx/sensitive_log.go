package zapx

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
)

type SensitiveLogCore struct {
	zapcore.Core
}

func (c SensitiveLogCore) Write(entry zapcore.Entry, fds []zapcore.Field) error {
	for _, fd := range fds {
		if fd.Key == "phone" {
			phone := fd.String
			fd.String = phone[:3] + "****" + phone[7:] // 对手机号脱敏
		}
		if strings.Contains(fd.Key, "password") || strings.Contains(fd.Key, "pwd") {
			fd.String = "********" // 对密码脱敏
		}
	}
	return c.Core.Write(entry, fds) // 装饰器模式
}

func PhoneMask(key string, phone string) zap.Field { // 对手机号脱敏
	return zap.Field{
		Key:    key,
		String: phone[:3] + "****" + phone[7:],
	}
}

func PasswordMask(key string, password string) zap.Field { // 对手机号脱敏
	return zap.Field{
		Key:    key,
		String: "********",
	}
}
