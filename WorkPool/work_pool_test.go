package WorkPool

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestWorkPool(t *testing.T) {
	// 创建初始3个worker，任务队列容量10
	pool := NewWorkerPool(3, 10)

	// 提交10个任务
	for i := 0; i < 10; i++ {
		i := i
		err := pool.Submit(func() {
			time.Sleep(time.Second)
			fmt.Printf("Task %d processed\n", i)
		})
		assert.NoError(t, err)
	}
	time.Sleep(500 * time.Millisecond)
	pool.Stop()
	//// 动态调整worker数量到5
	//pool.Resize(5)
	// 停止所有worker

	//time.Sleep(5 * time.Second)
}
