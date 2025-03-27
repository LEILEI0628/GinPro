package sms

import "context"

type Service interface { // 短信服务不要和业务强耦合（如强耦合验证码）
	Send(ctx context.Context, tpl string, args []string, numbers ...string) error
}
