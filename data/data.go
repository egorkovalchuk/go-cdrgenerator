package data

import (
	"fmt"
	"math/rand"
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
	Duration  int
	BRT       string
	BRT_port  int
	DateRange struct {
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
}

//Структура строки пула
type RecTypePool struct {
	Msisdn     string
	IMSI       string
	CallsCount int
}

type PoolSubs []RecTypePool

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

//Заполнение количествово выбора на абонента
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

//Преобразование вещественного и строку
func FloatToString(input_num float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(input_num, 'f', 6, 64)
}

//Формирование записи для CDR
//Формируется из шаблона с заменой
func CreateCDRRecord(RecordMsisdn RecTypePool, date time.Time, RecordType RecTypeRatioType, cfg string) string {
	// Номер записи, добавить генерацию
	rec_number := 1
	//TasksType.CDR_pattern
	CDR_pattern := cfg

	CDR_pattern = strings.Replace(CDR_pattern, "{rec_type}", RecordType.Name, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{type_code}", RecordType.TypeCode, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{type_ser}", RecordType.TypeSer, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{imsi}", RecordMsisdn.IMSI, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{msisdn}", RecordMsisdn.Msisdn, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{rec_number}", strconv.Itoa(rec_number), 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{datetime}", date.Format("20060201030405"), 1)

	return CDR_pattern
}

// map c mutex
type Counters struct {
	mx sync.Mutex
	m  map[string]int
}

func NewCounters() *Counters {
	return &Counters{
		m: make(map[string]int),
	}
}

func (c *Counters) Load(key string) int {
	c.mx.Lock()
	val, _ := c.m[key]
	c.mx.Unlock()
	return val
}

func (c *Counters) Store(key string, value int) {
	c.mx.Lock()
	c.m[key] = value
	c.mx.Unlock()
}

func (c *Counters) Inc(key string) {
	c.mx.Lock()
	c.m[key]++
	c.mx.Unlock()
}
