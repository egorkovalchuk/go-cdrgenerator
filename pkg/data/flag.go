package data

import (
	"sync"
)

// map c mutex
// для контроля потока записи. Мутекс для избегания блокировок
type FlagType struct {
	mx sync.Mutex
	m  map[string]int
}

// Конструктор для типа данных Flag
func NewFlag() *FlagType {
	return &FlagType{
		m: make(map[string]int),
	}
}

// Получить значение
func (c *FlagType) Load(key string) int {
	c.mx.Lock()
	val := c.m[key]
	c.mx.Unlock()
	return val
}

// Загрузить значение
func (c *FlagType) Store(key string, value int) {
	c.mx.Lock()
	c.m[key] = value
	c.mx.Unlock()
}
