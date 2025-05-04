package saramax

import (
	"encoding/json"
	"github.com/IBM/sarama"
	loggerx "github.com/LEILEI0628/GinPro/middleware/logger"
)

type HandlerV1[T any] func(msg *sarama.ConsumerMessage, t T) error

type Handler[T any] struct {
	l  loggerx.Logger
	fn func(msg *sarama.ConsumerMessage, t T) error
}

func NewHandler[T any](l loggerx.Logger, fn func(msg *sarama.ConsumerMessage, t T) error) *Handler[T] {
	return &Handler[T]{
		l:  l,
		fn: fn,
	}
}

func (h Handler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h Handler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h Handler[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgs := claim.Messages()
	for msg := range msgs {
		var t T
		err := json.Unmarshal(msg.Value, &t)
		if err != nil {
			h.l.Error("反序列化消息失败",
				loggerx.Error(err),
				loggerx.String("topic", msg.Topic),
				loggerx.Int64("partition", int64(msg.Partition)),
				loggerx.Int64("offset", msg.Offset))
			continue
		}
		// 在此执行重试
		for i := 0; i < 3; i++ {
			err = h.fn(msg, t)
			if err == nil {
				break
			}
			h.l.Error("处理消息失败",
				loggerx.Error(err),
				loggerx.String("topic", msg.Topic),
				loggerx.Int64("partition", int64(msg.Partition)),
				loggerx.Int64("offset", msg.Offset))
		}

		if err != nil {
			h.l.Error("处理消息失败-重试次数上限",
				loggerx.Error(err),
				loggerx.String("topic", msg.Topic),
				loggerx.Int64("partition", int64(msg.Partition)),
				loggerx.Int64("offset", msg.Offset))
		} else {
			session.MarkMessage(msg, "")
		}
	}
	return nil
}
