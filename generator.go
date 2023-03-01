package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egorkovalchuk/go-cdrgenerator/data"
	"github.com/fiorix/go-diameter/diam"
	"github.com/fiorix/go-diameter/diam/avp"
	"github.com/fiorix/go-diameter/diam/datatype"
	"github.com/fiorix/go-diameter/diam/dict"
	"github.com/fiorix/go-diameter/diam/sm"
	"github.com/fiorix/go-diameter/diam/sm/smpeer"
)

//Power by  Egor Kovalchuk

const (
	logFileName = "generator.log"
	pidFileName = "generator.pid"
	versionutil = "0.2"
)

var (
	//конфиг
	cfg data.Config

	// режим работы сервиса(дебаг мод)
	debugm bool

	// ошибки
	err error

	// режим работы сервиса
	startdaemon bool

	// запрос версии
	version bool

	//Запись в фаил
	tofile bool

	// для выбора типа соединения
	brt      bool
	brtcamel bool

	//Каналы для управления и передачи информации
	CDRChannel     = make(chan string)
	ProcessChannel = make(chan string)
	ErrorChannel   = make(chan error)
	//Признак запуска дополнительного потока
	Flag = data.NewFlag()
	//Скорость потока
	CDRPerSec = data.NewCounters()
	//Срез для каналов
	CDRChanneltoFileUni = make(map[string](chan string))
	CDRChanneltoBRTUni  = make(map[string](chan string))
	//Статистика записи
	CDRRecCount     = data.NewCounters()
	CDRFileCount    = data.NewCounters()
	CDRRecTypeCount = data.NewRecTypeCounters()

	//Канал для Диметра коннект к БРТ
	BrtDiamChannelAnswer = make(chan struct{}, 1000)
	BrtDiamChannel       = make(chan struct{}, 1000)

	//Не используется(в коде закоменчено)
	CDRChanneltoFile     = make(chan string)
	CDRRoamChanneltoFile = make(chan string)

	m sync.Mutex

/*
Vesion 0.2
add Diameter connection to Nexign NWM produtcs (3GPP Diameter Credit-Control Application)
CCR/CCA type request "Event"
*/

)

func main() {

	//start program
	var argument string
	/*var progName string

	progName = os.Args[0]*/

	if os.Args != nil && len(os.Args) > 1 {
		argument = os.Args[1]
	} else {
		data.HelpStart()
		return
	}

	if argument == "-h" {
		data.HelpStart()
		return
	}

	flag.BoolVar(&debugm, "debug", false, "a bool")
	flag.BoolVar(&startdaemon, "d", false, "a bool")
	flag.BoolVar(&tofile, "file", false, "a bool")
	flag.BoolVar(&version, "v", false, "a bool")
	flag.BoolVar(&brt, "brt", false, "Connect to BRT Diameter")
	flag.BoolVar(&brtcamel, "brtcamel", false, "Connect to BRT Camel")
	// for Linux compile
	// Для использования передачи системных сигналов
	stdaemon := flag.Bool("s", false, "a bool") // для передачи
	// --for Linux compile
	flag.Parse()

	// Открытие лог файла
	// ротация не поддерживается в текущей версии
	filer, err := os.OpenFile(logFileName, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(filer)
	log.Println("- - - - - - - - - - - - - - -")

	ProcessDebug("Start with debug mode")

	if startdaemon {

		log.Println("Start daemon mode")
		if debugm {
			log.Println("Start with debug mode")
		}

		fmt.Println("Start daemon mode")
	}

	if version {
		fmt.Println("Version utils " + versionutil)
		return
	}

	//load conf
	readconf(&cfg, "config.json")

	//Если не задан параметр используем дефольтное значение
	if cfg.Common.Duration == 0 {
		log.Println("Script use default duration - 14400 sec")
		cfg.Common.Duration = 14400
	}

	// Обнуляем счетчик и инициализируем
	for _, task := range cfg.Tasks {
		// Иницмализация счетчика
		CDRPerSec.Store(task.Name, 0)
		// Инициализация флага запуска дополнительной горутины
		Flag.Store(task.Name, 0)
		//Инициализация каналов
		CDRChanneltoFileUni[task.Name] = make(chan string)
		CDRChanneltoBRTUni[task.Name] = make(chan string)

		//Добавлено для тестов, по идее использовать CDRChanneltoBRTUni
		BrtDiamChannelAnswer = make(chan struct{}, 1000)
		BrtDiamChannel = make(chan struct{}, 1000)

		// Заполнение интервалов для радндомайзера
		// Инициализация среза для полсчета типов
		task.RecTypeRatio[0].RangeMax = task.RecTypeRatio[0].Rate
		task.RecTypeRatio[0].RangeMin = 0
		for i := 1; i < len(task.RecTypeRatio); i++ {
			task.RecTypeRatio[i].RangeMin = task.RecTypeRatio[i-1].RangeMax
			task.RecTypeRatio[i].RangeMax = task.RecTypeRatio[i].Rate + task.RecTypeRatio[i].RangeMin
			Flag.Store(task.Name+" "+task.RecTypeRatio[i].Name, 0)
			CDRRecTypeCount.AddMap(task.Name, task.RecTypeRatio[i].Name, 0)
		}

	}

	// запуск горутины записи в лог
	go LogWriteForGoRutine(ErrorChannel)
	// Запуск мониторинга
	go Monitor()

	//Определяем ветку
	if startdaemon || *stdaemon {

		//processinghttp(&cfg, debugm)

		log.Println("daemon terminated")

	} else if brt {
		StartDiameterClient()
	} else {

		StartSimpleMode()

	}
	fmt.Println("Done")
	return

}

// Нештатное завершение при критичной ошибке
func ProcessError(err error) {
	fmt.Println(err)
	os.Exit(2)
}

func readconf(cfg *data.Config, confname string) {
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

// StartSimpleMode запуск в режиме скрипта
func StartSimpleMode() {
	// запускаем отдельные потоки родительские потоки для задач из конфига
	// родительские породождают дочерние по формуле
	// при завершении времени останавливают дочерние и сами
	// Добавить сигнал остановки

	//Основной цикл
	for _, thread := range cfg.Tasks {
		if thread.DatapoolCsvFile == "" {
			log.Println("Please, set the file name specified for" + thread.Name)
		} else {
			f, err := os.Open(thread.DatapoolCsvFile)
			if err != nil {
				log.Println("Unable to read input file "+thread.DatapoolCsvFile, err)
				log.Println("Thread " + thread.Name + " not start")
			} else {
				defer f.Close()
				//Вынести в глобальные?
				ProcessDebug("Start load" + thread.DatapoolCsvFile)

				// read csv values using csv.Reader
				csvReader := csv.NewReader(f)
				csvReader.Comma = ';'
				csv, err := csvReader.ReadAll()

				var PoolList data.PoolSubs
				PoolList = PoolList.CreatePoolList(csv, thread)

				if err != nil {
					log.Println(err)
				} else {
					ProcessDebug("Last record ")
					ProcessDebug(PoolList[len(PoolList)-1])

					log.Println("Load " + strconv.Itoa(len(PoolList)) + " records")
					log.Println("Start thread for " + thread.Name)
					// Если пишем в фаил
					// Запускать потоки записи по количеству путей? Не будет ли пересечение по именам файлов
					if tofile {
						/*if thread.Name == "local" {
							go StartFileCDR(thread.PathsToSave[0]+thread.Template_save_file, CDRChanneltoFile)
						} else {
							go StartFileCDR(thread.PathsToSave[0]+thread.Template_save_file, CDRRoamChanneltoFile)
						}*/
						go StartFileCDR(thread, CDRChanneltoFileUni[thread.Name])
					}

					go StartTask(PoolList, thread, true)
				}

			}
		}
	}

	log.Println("Start schelduler")
	time.Sleep(time.Duration(cfg.Common.Duration) * time.Second)
	for _, thread := range cfg.Tasks {
		log.Println("All load " + strconv.Itoa(CDRRecCount.Load(thread.Name)) + " records for " + thread.Name)
		if tofile {
			log.Println("Save " + strconv.Itoa(CDRFileCount.Load(thread.Name)) + " files for " + thread.Name)
		}
	}
	log.Println("End schelduler")

}

// Функция контроля рейтов
// Считать ли по рек тайпам?
func Monitor() {
	heartbeat := time.Tick(1 * time.Second)
	heartbeat10 := time.Tick(10 * time.Second)

	var CDR int

	time.Sleep(5 * time.Second)

	for {
		select {
		case <-heartbeat:
			for _, thread := range cfg.Tasks {
				CDR = CDRPerSec.Load(thread.Name)
				ProcessDebug("Speed task " + thread.Name + " " + strconv.Itoa(CDR) + " op/s")

				if CDR < thread.CallsPerSecond {
					Flag.Store(thread.Name, 1)
				}
				CDRRecCount.IncN(thread.Name, CDR)
				CDRPerSec.Store(thread.Name, 0)
			}
		case <-heartbeat10:
			for _, thread := range cfg.Tasks {
				log.Println("Load " + strconv.Itoa(CDRRecCount.Load(thread.Name)) + " records for " + thread.Name)
				if tofile {
					log.Println("Save " + strconv.Itoa(CDRFileCount.Load(thread.Name)) + " files for " + thread.Name)
				}
				for _, t := range thread.RecTypeRatio {
					log.Println("Load " + thread.Name + " calls type " + t.Name + " " + CDRRecTypeCount.LoadString(thread.Name, t.Name))
				}
			}
		}
	}
}

// Горутина формирования CDR
func StartTask(PoolList []data.RecTypePool, cfg data.TasksType, FirstStart bool) {

	var PoolIndex int
	var PoolIndexMax int
	var CDR int
	var RecTypeIndex int
	var tmp int
	PoolIndex = 0
	PoolIndexMax = len(PoolList) - 1

	for {
		// Порождать доп процессы может только первый процесс
		// Уменьшает обращение с блокировками переменной флаг
		if FirstStart {
			if Flag.Load(cfg.Name) == 1 {
				log.Println("Start new thead " + cfg.Name)
				go StartTask(PoolList, cfg, false)
				Flag.Store(cfg.Name, 0)
			}
		}

		CDR = CDRPerSec.Load(cfg.Name)
		if CDR < cfg.CallsPerSecond {
			// Сброс счетчика
			if PoolIndex >= PoolIndexMax {
				PoolIndex = 0
			}

			//Вынес в формирование файла/передачи в БРТ
			//для расчета постфактум
			//CDRPerSec.Inc(cfg.Name)
			PoolIndex++

			tmp = PoolList[PoolIndex].CallsCount

			if tmp != 0 {

				PoolList[PoolIndex].CallsCount = tmp - 1

				RecTypeIndex = data.RandomRecType(cfg.RecTypeRatio, rand.Intn(100))
				CDRRecTypeCount.Inc(cfg.Name, cfg.RecTypeRatio[RecTypeIndex].Name)

				rr := data.CreateCDRRecord(PoolList[PoolIndex], time.Now(), cfg.RecTypeRatio[RecTypeIndex], cfg.CDR_pattern)

				/*if cfg.Name == "local" {
					CDRChanneltoFile <- rr
				} else {
					CDRRoamChanneltoFile <- rr
				}*/
				//m.Lock()
				CDRChanneltoFileUni[cfg.Name] <- rr
				//m.Unlock()
				if err != nil {
					ErrorChannel <- err
				}
			}
		}

	}

}

// Запись в Фаил
// Исправил на канал для чтения
func StartFileCDR(task data.TasksType, InputString chan string) {
	//func StartFileCDR(task data.TasksType, InputString <-chan string) {
	// Запись в разные каталоги
	var DirNum int
	var DirNumLen int
	DirNumLen = len(task.PathsToSave)

	f, err := os.Create(strings.Replace(task.PathsToSave[0]+task.Template_save_file, "{date}", time.Now().Format("20060201030405"), 1))

	if err != nil {
		ErrorChannel <- err
	}
	if debugm {
		log.Println("Start write " + f.Name())
	}
	defer f.Close()

	//heartbeat := time.Tick(1 * time.Second)
	//Переписать на создание нового файла каждую Х секнду
	heartbeat := time.Tick(2 * time.Second)

	for {
		//for str := range InputString {
		select {
		case <-heartbeat:
			f.Close()
			// Добавить директорию на выбор
			DirNum = rand.Intn(DirNumLen)
			f, err = os.Create(strings.Replace(task.PathsToSave[DirNum]+task.Template_save_file, "{date}", time.Now().Format("20060201030405"), 1))

			if err != nil {
				ErrorChannel <- err
			}

			//Переместить в горутину
			ProcessDebug("Start write " + f.Name())

			CDRFileCount.Inc(task.Name)
			defer f.Close() //Закрыть фаил при нешаттном завершении
		default:

			str := <-InputString
			//Перенес из генерации, в одном потоке будет работать быстрее
			CDRPerSec.Inc(task.Name)

			_, err = f.WriteString(str + "\n")

			if err != nil {
				ErrorChannel <- err
			}
		}
	}
}

// Чтение из потока файлов (работа без генератора)
func StartTransferCDR(FileName string, InputString <-chan string) {

}

// Запуск потоков подключения к БРТ
func StartDiameterClient() {

	var brt_adress []datatype.Address
	for _, ip := range cfg.Common.BRT {
		brt_adress = append(brt_adress, datatype.Address(net.ParseIP(ip)))
	}

	//прописываем конфиг
	ProcessDebug("Load Diameter config")

	diam_cfg := &sm.Settings{
		OriginHost:       datatype.DiameterIdentity("client"),
		OriginRealm:      datatype.DiameterIdentity("go-diameter"),
		VendorID:         data.PETER_SERVICE_VENDOR_ID,
		ProductName:      "CDR-generator",
		OriginStateID:    datatype.Unsigned32(time.Now().Unix()),
		FirmwareRevision: 1,
		//HostIPAddresses:  brt_adress,
	}

	// Create the state machine (it's a diam.ServeMux) and client.
	mux := sm.New(diam_cfg)
	ProcessDebug(mux.Settings())

	// Запуск потока записи ошибок в лог
	go DiamPrintErrors(mux.ErrorReports())

	ProcessDebug("Load Diameter dictionary")

	// Load our custom dictionary on top of the default one.
	/*err := dict.Default.Load(bytes.NewReader([]byte(data.HelloDictionary)))
	if err != nil {
		log.Fatal(err)
	}*/

	ProcessDebug("Load Diameter client")

	cli := &sm.Client{
		Dict:               dict.Default,
		Handler:            mux,
		MaxRetransmits:     3,
		RetransmitInterval: time.Second,
		EnableWatchdog:     true,
		WatchdogInterval:   5 * time.Second,

		VendorSpecificApplicationID: []*diam.AVP{
			diam.NewAVP(avp.VendorSpecificApplicationID, avp.Mbit, 0, &diam.GroupedAVP{
				AVP: []*diam.AVP{
					diam.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(4)),                   //16777302
					diam.NewAVP(avp.VendorID, avp.Mbit, 0, datatype.Unsigned32(data.PETER_SERVICE_VENDOR_ID)), //10415
				},
			}),
		},
	}
	// Set message handlers.
	mux.Handle("CCA", handleCCA(BrtDiamChannelAnswer))

	// networkType - protocol type tcp/sctp
	// Пока прописал в явном виде один адрес
	/*connect := func() (diam.Conn, error) {
		return dial(cli, "10.199.112.194:4868", "", "", false, "tcp")
	}*/

	cli.EnableWatchdog = true

	brt_connect := make([]diam.Conn, len(brt_adress))

	for _, init_connect := range cfg.Common.BRT {
		ProcessDebug(init_connect)

		var err error

		log.Println("Connecting clients...")
		for i := 0; i < len(brt_adress); i++ {
			brt_connect[i], err = Dial(cli, init_connect+":"+strconv.Itoa(cfg.Common.BRT_port), "", "", false, "tcp")
			if err != nil {
				log.Println("Connect error ")
				log.Fatal(err)
			}
			defer brt_connect[i].Close()
		}
		log.Println("Done. Sending messages...")

	}

	//StartDiameterClientThread(connect, diam_cfg, BrtDiamChannel)

}

// Кусок для диаметра
// Определение шифрование соединения
func Dial(cli *sm.Client, addr, cert, key string, ssl bool, networkType string) (diam.Conn, error) {
	if ssl {
		return cli.DialNetworkTLS(networkType, addr, cert, key)
	}
	return cli.DialNetwork(networkType, addr)
}

//Обработчик-ответа Диаметра
func handleCCA(done chan struct{}) diam.HandlerFunc {
	ok := struct{}{}
	return func(c diam.Conn, m *diam.Message) {
		done <- ok
		log.Println(m)
	}
}

//вынести. если будет использоваться
type dialFunc func() (diam.Conn, error)

func StartDiameterClientThread(df dialFunc, cfg *sm.Settings, done chan struct{}) {
	var err error
	//Один коннект

	c := make([]diam.Conn, 1)
	log.Println("Connecting clients...")
	for i := 0; i < 1; i++ {
		c[i], err = df() // Dial and do CER/CEA handshake.
		if err != nil {
			log.Println("Connect error ")
			log.Fatal(err)
		}
		defer c[i].Close()
	}
	log.Println("Done. Sending messages...")

	for _, cli := range c {
		SendCCREvent(cli, cfg, BrtDiamChannel)
	}
}

var eventRecord = datatype.Unsigned32(1)

func SendCCREvent(c diam.Conn, cfg *sm.Settings, in <-chan struct{}) {

	// заменить на просто вывод в лог
	meta, ok := smpeer.FromContext(c.Context())
	if !ok {
		log.Fatal("Client connection does not contain metadata")
	}
	var err error
	var m *diam.Message
	for i := 0; i < 1; i++ {
		m = diam.NewRequest(diam.Accounting, 0, c.Dictionary())
		m.NewAVP(avp.SessionID, avp.Mbit, 0,
			datatype.UTF8String(strconv.Itoa(i)))
		m.NewAVP(avp.OriginHost, avp.Mbit, 0, cfg.OriginHost)
		m.NewAVP(avp.OriginRealm, avp.Mbit, 0, cfg.OriginRealm)
		m.NewAVP(avp.DestinationRealm, avp.Mbit, 0, meta.OriginRealm)
		m.NewAVP(avp.AccountingRecordType, avp.Mbit, 0, eventRecord)
		m.NewAVP(avp.AccountingRecordNumber, avp.Mbit, 0,
			datatype.Unsigned32(i))
		m.NewAVP(avp.DestinationHost, avp.Mbit, 0, meta.OriginHost)
		if _, err = m.WriteTo(c); err != nil {
			log.Fatal(err)
		}
	}
}

// Поток телнета
// два типа каналов CDR и закрытие/переоткрытие
// +надо сделать keepalive
func StartDiameterTelnet() {

}

// Поток телнета Кемел
// два типа каналов CDR и закрытие/переоткрытие
// +надо сделать keepalive
func StartCamelTelnet() {

}

//Просто запись в лог
func LogWrite(err error) {
	if startdaemon {
		log.Println(err)
	} else {
		fmt.Println(err)
	}
}

// Запись ошибок из горутин
func LogWriteForGoRutine(err <-chan error) {
	for err := range err {
		log.Println(err)
	}
}

// Запись ошибок из горутин для диаметра
func DiamPrintErrors(ec <-chan *diam.ErrorReport) {
	for err := range ec {
		log.Println(err)
	}
}

// Запись в лог при включенном дебаге
func ProcessDebug(logtext interface{}) {
	if debugm {
		log.Println(logtext)
	}
}
