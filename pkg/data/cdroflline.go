package data

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Сткуртура для массива пост обработки
type TypeBrtOfflineCdr struct {
	RecPool  RecTypePool
	CDRtime  time.Time
	Ratio    RecTypeRatioType
	TaskName string
	// для кемел
	DstMsisdn string
	// duration
	Lac  int
	Cell int
}

// Массив для работы с кешем запросов
type BrtOfflineCdr struct {
	mx         sync.RWMutex
	CDROffline map[string](TypeBrtOfflineCdr)
}

// Конструктор для типа данных BrtOfflineCdr для кеша
func NewCDROffline() *BrtOfflineCdr {
	return &BrtOfflineCdr{
		CDROffline: make(map[string](TypeBrtOfflineCdr)),
	}
}

// Получить значение
func (c *BrtOfflineCdr) Load(key string) TypeBrtOfflineCdr {
	c.mx.RLock()
	val := c.CDROffline[key]
	c.mx.RUnlock()
	return val
}

// Загрузить значение
func (c *BrtOfflineCdr) Store(key string, value TypeBrtOfflineCdr) {
	c.mx.Lock()
	c.CDROffline[key] = value
	c.mx.Unlock()
}

// Удалить значение
func (c *BrtOfflineCdr) Delete(key string) {
	c.mx.Lock()
	delete(c.CDROffline, key)
	c.mx.Unlock()
}

// Рандомное значение
func (c *BrtOfflineCdr) Random() (rr string) {
	c.mx.Lock()
	k := rand.Intn(len(c.CDROffline))
	for d, r1 := range c.CDROffline {
		if k == 0 {
			c.mx.Unlock()
			return d + fmt.Sprint(r1)
		}
		k--
	}
	c.mx.Unlock()
	return rr
}
