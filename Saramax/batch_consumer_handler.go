package saramax

import (
	"context"
	"encoding/json"
	"github.com/IBM/sarama"
	loggerx "github.com/LEILEI0628/GinPro/middleware/logger"
	"time"
)

type BatchHandler[T any] struct {
	l  loggerx.Logger
	fn func(msgs []*sarama.ConsumerMessage, ts []T) error
	// 用option模式来设置batchSize和duration
	batchSize     int
	batchDuration time.Duration
}

func NewBatchHandler[T any](l loggerx.Logger, fn func(msgs []*sarama.ConsumerMessage, ts []T) error) *BatchHandler[T] {
	return &BatchHandler[T]{l: l, fn: fn, batchDuration: time.Second, batchSize: 10}
}

func (b *BatchHandler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (b *BatchHandler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (b *BatchHandler[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgsCh := claim.Messages()
	batchSize := b.batchSize
	for {
		ctx, cancel := context.WithTimeout(context.Background(), b.batchDuration)
		done := false
		msgs := make([]*sarama.ConsumerMessage, 0, batchSize)
		ts := make([]T, 0, batchSize)
		for i := 0; i < batchSize && !done; i++ {
			select {
			case <-ctx.Done():
				done = true
			case msg, ok := <-msgsCh:
				if !ok {
					cancel()
					// 代表消费者被关闭
					return nil
				}
				var t T
				err := json.Unmarshal(msg.Value, &t)
				if err != nil {
					b.l.Error("反序列化失败",
						loggerx.Error(err),
						loggerx.String("topic", msg.Topic),
						loggerx.Int64("partition", int64(msg.Partition)),
						loggerx.Int64("offset", msg.Offset))
					continue
				}
				msgs = append(msgs, msg)
				ts = append(ts, t)
			}
		}
		cancel()
		if len(msgs) == 0 {
			continue
		}
		err := b.fn(msgs, ts)
		if err != nil {
			b.l.Error("调用业务批量接口失败",
				loggerx.Error(err))
			// 记录整个批次

			// 继续往前消费
		}
		for _, msg := range msgs {
			// 标记最后一个也可以，这样写最安全
			session.MarkMessage(msg, "")
		}
	}
}
