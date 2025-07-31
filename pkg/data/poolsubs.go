package data

// Структура строки пула
type RecTypePool struct {
	Msisdn     string
	IMSI       string
	CallsCount int
}

type PoolSubs []RecTypePool

// Заполнение массива для последующей генерации нагрузки
func (p PoolSubs) CreatePoolList(data [][]string, Task TasksType) PoolSubs {
	var PoolList PoolSubs
	for i, line := range data {
		if i > 0 && checkRowTypes(line) { // omit header line
			var rec RecTypePool
			rec.Msisdn = "7" + line[0]
			rec.IMSI = line[1]
			rec.CallsCount = Task.GenCallCount()
			PoolList = append(PoolList, rec)
		}
	}
	return PoolList
}

func (p PoolSubs) ReinitializationPoolList(Task TasksType) {
	for i := 0; i < len(p); i++ {
		p[i].CallsCount = Task.GenCallCount()
	}
}
