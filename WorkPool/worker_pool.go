package WorkPool

import (
	"errors"
	"sync"
	"time"
)

type Task func()

type WorkerPool struct {
	tasks          chan Task     // 任务channel，用于传递待处理的任务（缓冲区大小控制最大堆积任务量，当队列满时Submit会阻塞）
	stop           chan struct{} // 停止信号channel，用于通知worker停止工作
	mu             sync.Mutex
	workerCount    int            // 当前实际工作的任务数量
	desiredWorkers int            // 期望的worker数量（用于动态调整）
	wg             sync.WaitGroup // 添加 WaitGroup 跟踪活跃的 worker
}

func NewWorkerPool(initialWorkers, taskQueueSize int) *WorkerPool {
	wp := &WorkerPool{
		tasks: make(chan Task, taskQueueSize),
		stop:  make(chan struct{}),
	}
	wp.Resize(initialWorkers) // 初始化worker数量
	return wp
}

// Resize 调整工作协程数量
func (wp *WorkerPool) Resize(n int) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if n < 0 {
		n = 0
	}
	wp.desiredWorkers = n

	// 如果当前worker数量不足，启动新的worker
	for wp.workerCount < wp.desiredWorkers {
		wp.workerCount++ // 增加计数器
		go wp.worker()   // 增加协程时立即创建新worker，减少协程时，多余的worker会在完成任务后自动退出
	}
}

func (wp *WorkerPool) worker() {
	defer func() { // worker退出时减少计数器
		wp.mu.Lock()
		wp.workerCount--
		wp.mu.Unlock()
	}()

	for {
		wp.mu.Lock()
		desired := wp.desiredWorkers
		current := wp.workerCount
		wp.mu.Unlock()

		if current > desired { // 如果当前worker数量超过期望值，立即退出
			return
		}

		select { // 如果还有空闲期望值
		case task, ok := <-wp.tasks:
			if !ok { // channel关闭
				return
			}
			// 启动新任务时：
			func() {
				wp.wg.Add(1)
				defer wp.wg.Done()
				task() // 执行任务
			}()
		default:
			select {
			case <-wp.stop: // wp停止
				return
			case <-time.After(time.Millisecond * 100):
				// 防止在无任务时CPU空转
			}
		}
	}
}

// Submit 提交任务到工作池（wp关闭时再Submit会报错）
func (wp *WorkerPool) Submit(task Task) error {
	select {
	case wp.tasks <- task: // 将任务发送到channel
		return nil
	case <-wp.stop: // worker pool已关闭
		return errors.New("worker pool已关闭")
	}
}

func (wp *WorkerPool) Stop() {
	// Stop()方法会关闭所有通道并等待worker退出
	// 停止后提交任务会返回错误
	close(wp.stop)
	wp.wg.Wait()    // 等待任务缓存队列中的所有任务都执行成功后再退出
	close(wp.tasks) // 添加此close后Submit会Panic，不添加会返回error
}
