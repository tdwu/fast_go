package fast_base

import (
	"strconv"
	"sync"
)

type TaskPool[T any] struct {
	Channel   chan T
	WGroup    sync.WaitGroup
	WorkerNum int
	count     int
}

func (that *TaskPool[T]) Build(num int, fc func(string, T)) chan T {
	that.WorkerNum = num
	that.WGroup.Add(num)
	that.Channel = make(chan T, 100)

	var iwg sync.WaitGroup
	iwg.Add(that.WorkerNum)

	for i := 0; i < that.WorkerNum; i++ {
		go func(n int) {
			iwg.Done()
			defer that.WGroup.Done()
			num := strconv.Itoa(n)
			//  fmt.Printf("【%d】 开始\n", n)
			for {
				r, ok := <-that.Channel
				if !ok {
					break
				}
				fc(num, r)
			}

			//	fmt.Printf("【%d】 结束\n", n)

		}(i)
	}
	// 防止goroutine没启动起来
	iwg.Wait()
	return that.Channel
}
func (that *TaskPool[T]) Add(t T) {
	that.count++
	// fmt.Printf("发送累计：%d \n", that.count)
	that.Channel <- t
}
func (that *TaskPool[T]) Wait() {
	// fmt.Printf("等待执行结束：%d \n", that.count)
	close(that.Channel)
	that.WGroup.Wait()
}
