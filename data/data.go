package data

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
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
	Duration        int
	BRT             []string
	BRT_port        int
	BRT_OriginHost  string
	BRT_OriginRealm string
	CAMEL           struct {
		Port         int    `json:"Port"`
		SMSCAddress  string `json:"SMSCAddress"`
		Camel_SCP_id string `json:"Camel_SCP_id"`
		XVLR         string `json:"XVLR"`
		ContryCode   string `json:"ContryCode"`
		OperatorCode string `json:"OperatorCode"`
	} `json:"CAMEL"`
	Report struct {
		Influx       bool   `json:"Influx"`
		LoginInflux  string `json:"LoginInflux"`
		PassInflux   string `json:"PassInflux"`
		InfluxServer string `json:"InfluxServer"`
		Region       string `json:"Region"`
	} `json:"Report"`
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
	// Имя задачи
	Name string `json:"Name,omitempty"`
	// рейт
	CallsPerSecond int `json:"calls_per_second"`
	// Тип записи
	RecTypeRatio []RecTypeRatioType `json:"rec_type_ratio"`
	// Два параметра определяющее количество звонков на абоненте с распределением по процентам
	CallsRange struct {
		Percentile []float64 `json:"percentile"`
		Range      []int     `json:"range"`
	} `json:"CallsRange"`
	// имя файла датапула
	DatapoolCsvFile string `json:"datapool_csv_file"`
	// Путь для сохранения файлов
	PathsToSave []string `json:"paths_to_save"`
	// Шаблон сохранения файла
	Template_save_file string `json:"template_save_file"`
	// Паттерн для для СДР
	CDR_pattern string `json:"cdr_pattern"`
	// Вызываемый абонент по умолчанию
	DefaultMSISDN_B string `json:"DefaultMSISDN_B"`
	// LAC,CELL по умолчанию
	DefaultLAC  int `json:"DefaultLAC"`
	DefaultCELL int `json:"DefaultCELL"`
	// Пул с перечнем LAC/CELL
	DatapoolCsvLac string `json:"datapool_csv_lac"`
}

// Тип структуры описания логического вызова, сервис кодов
type RecTypeRatioType struct {
	Record_type      string `json:"record_type"`
	Name             string `json:"name"`
	Rate             int    `json:"rate"`
	TypeService      string `json:"type_service"`
	TypeCode         string `json:"type_code"`
	ServiceContextId string `json:"service_context_id"`
	MeasureType      string `json:"measure"`
	RatingGroup      int    `json:"rating_group"`
	DefaultChan      string `json:"default"`
	RangeMin         int
	RangeMax         int
}

// Структура строки пула
type RecTypePool struct {
	Msisdn     string
	IMSI       string
	CallsCount int
}

type PoolSubs []RecTypePool

// Структура строки пула
type RecTypeLACPool struct {
	LAC  int
	CELL int
}

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

// Пишем логи через горутину
type LogStruct struct {
	t    string
	text interface{}
}

// Структура для хранения паттерна
type CDRPatternType struct {
	Pattern string
	MsisdnB string
}

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

// Вызов справки
func HelpStart() {
	fmt.Println("Use -d start deamon mode")
	fmt.Println("Use -s stop deamon mode")
	fmt.Println("Use -debug start with debug mode")
	fmt.Println("Use -file save cdr to files(Offline)")
	fmt.Println("Use -brt message(cdr) transmission by diameter to the billing server ")
	fmt.Println("Use -brtlist task list (local,roam)")
	fmt.Println("Use -camel for UP SCP Server(Camel protocol)")
	fmt.Println("Use -rm Delete all files in directories(Test optional)")
	fmt.Println("Debug option")
	fmt.Println("Use -slow_camel for send Camel message every 10 seconds")
}

// Заполнение массива для последующей генерации нагрузки
func (p PoolSubs) CreatePoolList(data [][]string, Task TasksType) PoolSubs {
	var PoolList PoolSubs
	for i, line := range data {
		if i > 0 { // omit header line
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

// Заполнение количествово выбора на абонента
func (p *TasksType) GenCallCount() int {
	perc := rand.Float64()
	arraylen := len(p.CallsRange.Percentile)
	var callcount int
	for i := 0; i < arraylen; i++ {
		if perc >= p.CallsRange.Percentile[i] && perc <= p.CallsRange.Percentile[i+1] {
			callcount = rand.Intn(p.CallsRange.Range[i+1])
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

// Преобразование вещественного и строку
func FloatToString(input_num float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(input_num, 'f', 6, 64)
}

// Формирование записи для CDR
// Формируется из шаблона с заменой
// Переменная DstMsisdn если не задана, то используется значение по умолчанию
func CreateCDRRecord(RecordMsisdn RecTypePool, date time.Time, RecordType RecTypeRatioType, cfg CDRPatternType, DstMsisdn string, lc RecTypeLACPool) (string, error) {
	// Номер записи, добавить генерацию
	rec_number := time.Now().Format("0201030405")
	//TasksType.CDR_pattern
	CDR_pattern := cfg.Pattern

	CDR_pattern = strings.Replace(CDR_pattern, "{rec_type}", RecordType.Record_type, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{type_code}", RecordType.TypeCode, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{type_ser}", RecordType.TypeService, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{imsi}", RecordMsisdn.IMSI, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{msisdn}", RecordMsisdn.Msisdn, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{rec_number}", rec_number, 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{datetime}", date.Format("20060102030405"), 1)
	if DstMsisdn != "" {
		CDR_pattern = strings.Replace(CDR_pattern, "{msisdnB}", DstMsisdn, 1)
	} else {
		CDR_pattern = strings.Replace(CDR_pattern, "{msisdnB}", cfg.MsisdnB, 1)
	}
	CDR_pattern = strings.Replace(CDR_pattern, "{lac_a}", strconv.Itoa(lc.CELL), 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{call_a}", strconv.Itoa(lc.LAC), 1)
	return CDR_pattern, nil
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

// Получить значение
func (c *Counters) Load(key string) int {
	c.mx.Lock()
	val, _ := c.m[key]
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

// Получить значение
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

// список тасков испольняемых для диаметра BRT
type ArgListType []string

func (i *ArgListType) String() string {
	return fmt.Sprint(*i)
}

func (i *ArgListType) Set(value string) error {
	for _, dt := range strings.Split(value, ",") {
		*i = append(*i, dt)
	}
	return nil
}

func (i *ArgListType) Get(value string) bool {
	for _, dt := range *i {
		if dt == value {
			return true
		}
	}
	return false
}

//Массив для работы с кешем запросов
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
	val, _ := c.CDROffline[key]
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

// Генерация номера
func RandomMSISDN(tsk string) string {
	if tsk == "roam" {
		// логика для международных
		return ""
	} else {
		return "79" + strconv.Itoa(rand.Intn(100)) + strconv.Itoa(rand.Intn(10000000))
	}
}

func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
