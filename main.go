package main

import (
	"fmt"
	"sync"
	"time"
)

// Semaphore - структура семафора с произвольным значением "счетчика"
// ресурсов
type Semaphore struct {
	// Семафор - абстрактный тип данных,
	// в нашем случае в основе его лежит канал
	sem chan int
	// Время ожидания основных операций с семафором, чтобы не
	// блокировать
	// операции с ним навечно (необязательное требование, зависит от
	// нужд программы)
	timeout time.Duration
}

// Acquire - метод захвата семафора
func (s *Semaphore) Acquire() error {
	select {
	case s.sem <- 0:
		return nil
	case <-time.After(s.timeout):
		return fmt.Errorf("Не удалось захватить семафор")
	}
}

// Release - метод освобождения семафора
func (s *Semaphore) Release() error {
	select {
	case _ = <-s.sem:
		return nil
	case <-time.After(s.timeout):
		return fmt.Errorf("Не удалось освободить семафор")
	}
}

// NewSemaphore - функция создания семафора
func NewSemaphore(counter int, timeout time.Duration) *Semaphore {
	return &Semaphore{
		sem:     make(chan int, counter),
		timeout: timeout,
	}
}

// создадим программу и проведем эксперимент многопоточного счета с семафорами =)
// в ряде запусков можно обнаружить что не всегда считается до указанного конечного числа
// но нас интересует процесс =)
func main() {
	const readRoutineCount int = 20 //количиство потоков для чтения
	const semaCount int = 7         //количиство пропускаемых потоков для чтения во время счета
	const writeRoutineCount int = 15 //количество потоков записи
	const endCount = 1000           //до скольки считаем
	const waitTime = 300            //задержка милисекунд (чтобы наблюдать как семафор пропускает часть рутин, а остальные ждут)
	s := NewSemaphore(semaCount, waitTime*time.Millisecond)
	counts := 0
	cch := make(chan int, readRoutineCount)
	exit := make(chan int)
	sin := &sync.Cond{L: &sync.Mutex{}}
	printMutex := &sync.Mutex{}

	fmt.Print("\033[2J") //Clear screen

	//создаем потоки для счета
	for i := 0; i < readRoutineCount; i++ {
		go func(index int) {
			immer := true
			for immer {

				//ждем когда можно будет что-то посчитать
				sin.L.Lock()
				sin.Wait()
				sin.L.Unlock()
				i := <-cch
				// Начинаем работать с разделяемыми данными,
				// поэтому пытаемся захватить семафор
				if err := s.Acquire(); err != nil {
					// Код, в случае если кто-то уже работает с разделяемыми
					// данными					
					continue
				}
				// Выполняем важную работу с разделяемым ресурсом
				counts += i
				if counts == endCount {
					immer = false
				}

				//чтобы не съезжало и не дребезжало блокируем и печатаем
				printMutex.Lock()
				fmt.Printf("\033[%d;%dH", index, 0) // Set cursor position
				fmt.Println()
				fmt.Printf("\033[%d;%dH", index, 0) // Set cursor position
				fmt.Printf("%d: %d\n", index, counts)
				printMutex.Unlock()

				// Освобождаем семафор
				if err := s.Release(); err != nil {
					// Код, в случае если по ошибке освобождаем свободный
					// семафор
				}

				time.Sleep(waitTime * time.Millisecond)
			}
			exit <- counts
			close(exit)
		}(i + 1)
	}
	//создаем потоки для записи
	for i := 0; i < writeRoutineCount; i++ {
		go func() {
			immer := true
			for immer {
				//проверяем что счет еще идет
				select {
				case <-exit:
					immer = false
				case <-time.After(time.Duration(waitTime * time.Millisecond)):
					cch <- 1
					sin.Signal()
				}
			}
		}()
	}

	res := <-exit
	time.Sleep(2 * time.Second)
	fmt.Print("\033[2J") //Clear screen
	fmt.Println("Итог:", res)
}
