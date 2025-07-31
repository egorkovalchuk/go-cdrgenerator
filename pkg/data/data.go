package data

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
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
		Influx        bool   `json:"Influx"`
		InfluxToken   string `json:"InfluxToken"`
		InfluxOrg     string `json:"InfluxOrg"`
		InfluxVersion int    `json:"InfluxVersion"`
		InfluxBucket  string `json:"InfluxBucket"`
		InfluxServer  string `json:"InfluxServer"`
		Region        string `json:"Region"`
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
	// временной лаг задерржки для равномерного формирования запросов
	Time_delay int
	// Переменная успешности закгрузки пула
	Pool_loading bool
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
type RecTypeLACPool struct {
	LAC  int
	CELL int
}

// Структура для хранения паттерна
type CDRPatternType struct {
	Pattern string
	MsisdnB string
}

// Функция для проверки типов данных в строке
func checkRowTypes(record []string) bool {
	// Проверка первого поля msisdn
	_, err := strconv.Atoi(record[0])
	if err != nil {
		return false
	}

	// Проверка первого поля imsi
	_, err = strconv.Atoi(record[1])
	if err != nil {
		return false
	}

	if len(record[0]) != 10 || len(record[1]) != 15 {
		return false
	}

	return true
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

func (cfg *Config) ReadTask(task_name string) TasksType {
	for _, dt := range cfg.Tasks {
		if dt.Name == task_name {
			return dt
		}
	}
	return TasksType{}
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
	RecTypeCount := len(RecType)

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
	CDR_pattern = strings.Replace(CDR_pattern, "{lac_a}", strconv.Itoa(lc.LAC), 1)
	CDR_pattern = strings.Replace(CDR_pattern, "{cell_a}", strconv.Itoa(lc.CELL), 1)
	return CDR_pattern, nil
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
