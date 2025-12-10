package data

import (
	"log"
	"strconv"
	"strings"
	"sync"
)

// map c mutex
// для контроля потока записи. Мутекс для избегания блокировок
type Counters struct {
	mx sync.Mutex
	m  map[string]int
}

// Конструктор для типа данных Counters
func NewCounters() *Counters {
	return &Counters{
		m: make(map[string]int),
	}
}

// Получить значение
func (c *Counters) Load(key string) int {
	c.mx.Lock()
	val := c.m[key]
	c.mx.Unlock()
	return val
}

// Загрузить значение
func (c *Counters) Store(key string, value int) {
	c.mx.Lock()
	c.m[key] = value
	c.mx.Unlock()
}

// Инкримент +1
func (c *Counters) Inc(key string) {
	c.mx.Lock()
	c.m[key]++
	c.mx.Unlock()
}

// Инкримент +N
func (c *Counters) IncN(key string, inc int) {
	c.mx.Lock()
	c.m[key] += inc
	c.mx.Unlock()
}

// Загрузка в лог
func (c *Counters) LoadRangeToLog(s string, log *log.Logger) {
	c.mx.Lock()
	for k, v := range c.m {
		log.Println(s + k + ": " + strconv.Itoa(v))
	}
	c.mx.Unlock()
}

// Загрузка в лог через функцию
func (c *Counters) LoadRangeToLogFunc(s string, f func(logtext interface{})) {
	c.mx.Lock()
	for k, v := range c.m {
		f(s + k + ": " + strconv.Itoa(v))
	}
	c.mx.Unlock()
}

// Возврат карты
func (c *Counters) LoadMapSpeed(tmp map[string]int, Name string, Region string, ReportStat chan string, f func(logtext interface{})) map[string]int {
	c.mx.Lock()
	for i, j := range c.m {
		ReportStat <- "cdr_" + Name + "_resp,region=" + Region + ",task_name=all,Resp_code=" + strings.ReplaceAll(i, " ", "_") + " speed=" + strconv.Itoa((j-tmp[i])/10)
		tmp[i] = j
	}
	defer c.mx.Unlock()
	return tmp
}

// map c mutex
// для контроля потока записи. Мутекс для избегания блокировок
type RecTypeCounters struct {
	mx sync.Mutex
	m  map[string]map[string]int
}

// Конструктор для типа данных Counters для расчетов по типам
func NewRecTypeCounters() *RecTypeCounters {
	return &RecTypeCounters{
		m: make(map[string]map[string]int),
	}
}

func (c *RecTypeCounters) AddMap(key1 string, key2 string, val int) map[string]map[string]int {
	_, ok := c.m[key1]
	if !ok {
		mm := make(map[string]int)
		c.m[key1] = mm
	}
	c.m[key1][key2] = val
	return c.m
}

// Получить значение
func (c *RecTypeCounters) Load(key1 string, key2 string) int {
	c.mx.Lock()
	val := c.m[key1][key2]
	c.mx.Unlock()
	return val
}

// Загрузить значение
func (c *RecTypeCounters) Store(key1 string, key2 string, value int) {
	c.mx.Lock()
	c.m[key1][key2] = value
	c.mx.Unlock()
}

// Инкримент +1
func (c *RecTypeCounters) Inc(key1 string, key2 string) {
	c.mx.Lock()
	// if c.m[key1] ==nil -отключено, так есть обязательная инициализация при старте
	c.m[key1][key2]++
	c.mx.Unlock()
}

func (c *RecTypeCounters) LoadString(key1 string, key2 string) string {
	c.mx.Lock()
	val, _ := c.m[key1][key2]
	c.mx.Unlock()
	return strconv.Itoa(val)
}
