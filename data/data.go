package data

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Common CommonType `json:"common"`
	Tasks  []TasksType
}

type CommonType struct {
	Duration   int
	BRT        []string
	BRT_port   int
	CAMEL_port int
	DateRange  struct {
		Start string `json:"start"`
		End   string `json:"end"`
		Freq  string `json:"freq"`
	} `json:"date_range"`
	RampUp struct {
		Time  int `json:"time"`
		Steps int `json:"steps"`
	} `json:"ramp_up"`
	OutputDuration int `json:"output_duration"`
}

type TasksType struct {
	Name               string             `json:"Name,omitempty"`
	CallsPerSecond     int                `json:"calls_per_second"`
	RecTypeRatio       []RecTypeRatioType `json:"rec_type_ratio"`
	Percentile         []float64          `json:"percentile"`
	CallsRange         []int              `json:"calls_range"`
	DatapoolCsvFile    string             `json:"datapool_csv_file"`
	PathsToSave        []string           `json:"paths_to_save"`
	Template_save_file string             `json:"template_save_file"`
	CDR_pattern        string             `json:"cdr_pattern"`
}

//Тип структуры описания логического вызова, сервис кодов
type RecTypeRatioType struct {
	Name     string `json:"name"`
	Rate     int    `json:"rate"`
	TypeSer  string `json:"type_ser"`
	TypeCode string `json:"type_code"`
	RangeMin int
	RangeMax int
}

//Структура строки пула
type RecTypePool struct {
	Msisdn     string
	IMSI       string
	CallsCount int
}

type PoolSubs []RecTypePool

func (cfg *Config) ReadConf(confname string) {
	file, err := os.Open(confname)
	if err != nil {
		ProcessError(err)
	}
	// Закрытие при нештатном завершении
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		ProcessError(err)
	}

	file.Close()

}

//Вызов справки
func HelpStart() {
	fmt.Println("Use -d start deamon mode")
	fmt.Println("Use -s stop deamon mode")
	fmt.Println("Use -debug start with debug mode")
	fmt.Println("Use -file save cdr to files")
}

//Заполнение массива для последующей генерации нагрузки
func (p PoolSubs) CreatePoolList(data [][]string, Task TasksType) PoolSubs {
	var PoolList PoolSubs
	for i, line := range data {
		if i > 0 { // omit header line
			var rec RecTypePool
			rec.Msisdn = line[0]
			rec.IMSI = line[1]
			rec.CallsCount = Task.GenCallCount()
			PoolList = append(PoolList, rec)
		}
	}
	return PoolList
}

// Заполнение количествово выбора на абонента
func (p *TasksType) GenCallCount() int {
	perc := rand.Float64()
	arraylen := len(p.Percentile)
	var callcount int
	for i := 0; i < arraylen; i++ {
		if perc >= p.Percentile[i] && perc <= p.Percentile[i+1] {
			callcount = rand.Intn(p.CallsRange[i+1])
			break
		}
	}
	return callcount
}

// Возвращение типа звонка по рандому
func RandomRecType(RecType []RecTypeRatioType, c int) int {
	var RecTypeCount int
	RecTypeCount = len(RecType)

	for i := 0; i < RecTypeCount; i++ {
		if RecType[i].RangeMin < c && RecType[i].RangeMax > c {
			return i
		}
	}
	return 0
}

//Преобразование вещественного и строку
func FloatToString(input_num float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(input_num, 'f', 6, 64)
}

//Формирование записи для CDR
//Формируется из шаблона с заменой
func CreateCDRRecord(RecordMsisdn RecTypePool, date time.Time, RecordType RecTypeRatioType, cfg string) string {
	// Номер записи, добавить генерацию
	rec_number := time.Now().Format("0201030405")
	//TasksType.CDR_pattern
	CDR_pattern := cfg

	CDR_pattern = strings.Replace(CDR_pattern, "{rec_type}", RecordType.Name, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{type_code}", RecordType.TypeCode, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{type_ser}", RecordType.TypeSer, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{imsi}", RecordMsisdn.IMSI, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{msisdn}", RecordMsisdn.Msisdn, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{rec_number}", rec_number, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{datetime}", date.Format("20060201030405"), 1)

	return CDR_pattern
}

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

// Записать значение
func (c *Counters) Load(key string) int {
	c.mx.Lock()
	val, _ := c.m[key]
	c.mx.Unlock()
	return val
}

//Загрузить значение
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

// Записать значение
func (c *FlagType) Load(key string) int {
	c.mx.Lock()
	val, _ := c.m[key]
	c.mx.Unlock()
	return val
}

//Загрузить значение
func (c *FlagType) Store(key string, value int) {
	c.mx.Lock()
	c.m[key] = value
	c.mx.Unlock()
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
	mm, ok := c.m[key1]
	if !ok {
		mm = make(map[string]int)
		c.m[key1] = mm
	}
	c.m[key1][key2] = val
	return c.m
}

// Записать значение
func (c *RecTypeCounters) Load(key1 string, key2 string) int {
	c.mx.Lock()
	val, _ := c.m[key1][key2]
	c.mx.Unlock()
	return val
}

//Загрузить значение
func (c *RecTypeCounters) Store(key1 string, key2 string, value int) {
	c.mx.Lock()
	c.m[key1][key2] = value
	c.mx.Unlock()
}

// Инкримент +1
func (c *RecTypeCounters) Inc(key1 string, key2 string) {
	c.mx.Lock()
	c.m[key1][key2]++
	c.mx.Unlock()
}

func (c *RecTypeCounters) LoadString(key1 string, key2 string) string {
	c.mx.Lock()
	val, _ := c.m[key1][key2]
	c.mx.Unlock()
	return strconv.Itoa(val)
}

// Нештатное завершение при критичной ошибке
func ProcessError(err error) {
	fmt.Println(err)
	os.Exit(2)
}
