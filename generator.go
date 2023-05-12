package main

import (
	"encoding/csv"
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
	"github.com/fiorix/go-diameter/v4/diam"
	"github.com/fiorix/go-diameter/v4/diam/avp"
	"github.com/fiorix/go-diameter/v4/diam/datatype"
	"github.com/fiorix/go-diameter/v4/diam/dict"
	"github.com/fiorix/go-diameter/v4/diam/sm"
	"github.com/fiorix/go-diameter/v4/diam/sm/smpeer"
)

//Power by  Egor Kovalchuk

const (
	logFileName = "generator.log"
	pidFileName = "generator.pid"
	versionutil = "0.2.1"
)

var (
	//конфиг
	global_cfg data.Config

	// режим работы сервиса(дебаг мод)
	debugm bool

	// ошибки
	err error

	// режим работы сервиса
	startdaemon bool

	// запрос версии
	version bool

	//Запись в фаил
	tofile     bool
	tofilelist data.ArgListType

	// для выбора типа соединения
	brt     bool
	camel   bool
	brtlist data.ArgListType

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
	CDRBRTCount     = data.NewCounters()

	//Канал для Диметра коннект к БРТ
	BrtDiamChannelAnswer = make(chan struct{}, 1000)
	BrtDiamChannel       = make(chan data.DiamCH, 1000)

	//Не используется(в коде закоменчено)
	CDRChanneltoFile     = make(chan string)
	CDRRoamChanneltoFile = make(chan string)

	m sync.Mutex

/*
Vesion 0.2
add Diameter connection to Nexign NWM produtcs (3GPP Diameter Credit-Control Application)
CCR/CCA type request "Event"
Vesion 0.2.1
Fix Bug
Vesion 0.3.0

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

	flag.BoolVar(&debugm, "debug", false, "Start with debug mode")
	flag.BoolVar(&startdaemon, "d", false, "a bool")
	flag.BoolVar(&version, "v", false, "Print version")
	flag.BoolVar(&brt, "brt", false, "Connect to BRT Diameter protocol")
	flag.BoolVar(&tofile, "file", false, "Start save CDR to file")
	flag.BoolVar(&camel, "camel", false, "Connect to BRT Camel protocol")
	flag.Var(&brtlist, "brtlist", "List of name task to work BRT")
	//flag.Var(&tofilelist, "filelist", "Connect to BRT Diameter protocol")
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
	defer filer.Close()

	log.SetOutput(filer)
	log.Println("- - - - - - - - - - - - - - -")
	log.Println("INFO: Start util")

	ProcessDebug("Start with debug mode")

	if startdaemon {

		log.Println("Start daemon mode")
		ProcessDebug("Start with debug mode")
		fmt.Println("Start daemon mode")
	}

	if version {
		fmt.Println("Version utils " + versionutil)
		return
	}

	//Чтение конфига
	global_cfg.ReadConf("config.json")

	InitVariables()

	// запуск горутины записи в лог
	go LogWriteForGoRutine(ErrorChannel)
	// Запуск мониторинга
	go Monitor()

	// Определяем ветку
	// Запускать горутиной с ожиданием процесса сигналов от ОС
	if startdaemon || *stdaemon {
		log.Println("daemon terminated")
	} else {
		StartSimpleMode()
	}
	fmt.Println("Done")
	return

}

// StartSimpleMode запуск в режиме скрипта
func StartSimpleMode() {
	// запускаем отдельные потоки родительские потоки для задач из конфига
	// родительские породождают дочерние по формуле
	// при завершении времени останавливают дочерние и сами
	// Добавить сигнал остановки

	// Вынесено из теста. Нужна ли отдельная точка входа для роуминга?
	if brt {
		// Запуск потока БРТ
		// Горутина запускается из функции, по количеству серверов
		// Поток идет только по правилу один хост один пир
		StartDiameterClient()
	}

	//Основной цикл
	for _, thread := range global_cfg.Tasks {
		if thread.DatapoolCsvFile == "" {
			log.Println("WARM: Please, set the file name specified for " + thread.Name)
		} else {
			f, err := os.Open(thread.DatapoolCsvFile)
			if err != nil {
				log.Println("ERROR: Unable to read input file "+thread.DatapoolCsvFile, err)
				log.Println("ERROR: Thread " + thread.Name + " not start")
			} else {
				defer f.Close()
				//Вынести в глобальные?
				ProcessDebug("Start load" + thread.DatapoolCsvFile)

				// read csv values using csv.Reader
				csvReader := csv.NewReader(f)
				// Разделитель CSV
				csvReader.Comma = ';'
				csv, err := csvReader.ReadAll()

				//Заполнение количества звонков на абонента
				var PoolList data.PoolSubs
				PoolList = PoolList.CreatePoolList(csv, thread)

				if err != nil {
					log.Println(err)
				} else {
					ProcessDebug("Last record ")
					ProcessDebug(PoolList[len(PoolList)-1])

					log.Println("INFO: Load " + strconv.Itoa(len(PoolList)) + " records")

					if brt && brtlist.Get(thread.Name) {
						log.Println("INFO: Start diameter thread for " + thread.Name)
						go StartTaskDiam(PoolList, thread, true)
					} else if tofile {
						// Если пишем в фаил, в начале запускаем потоки записи в фаил или коннекта к BRT
						// Запускать потоки записи по количеству путей? Не будет ли пересечение по именам файлов
						log.Println("INFO: Start file thread for " + thread.Name)
						go StartFileCDR(thread, CDRChanneltoFileUni[thread.Name])
						go StartTaskFile(PoolList, thread, true)
					} else {
						log.Println("INFO: Usage options not set " + thread.Name)
					}

				}

			}
		}
	}

	log.Println("INFO: Start schelduler")
	// Ждем выполнение таймаута
	// Добавить в дальнейшем выход по событию от системы
	time.Sleep(time.Duration(global_cfg.Common.Duration) * time.Second)

	for _, thread := range global_cfg.Tasks {
		log.Println("INFO: All load " + strconv.Itoa(CDRRecCount.Load(thread.Name)) + " records for " + thread.Name)
		if tofile {
			log.Println("INFO: Save " + strconv.Itoa(CDRFileCount.Load(thread.Name)) + " files for " + thread.Name)
		}
	}
	if brt {
		for _, ip := range global_cfg.Common.BRT {
			log.Println("INFO: Send " + strconv.Itoa(CDRBRTCount.Load(ip)) + " messages to " + ip)
		}
	}
	log.Println("INFO: End schelduler")

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
			for _, thread := range global_cfg.Tasks {
				CDR = CDRPerSec.Load(thread.Name)
				ProcessDebug("Speed task " + thread.Name + " " + strconv.Itoa(CDR) + " op/s")

				if CDR < thread.CallsPerSecond {
					Flag.Store(thread.Name, 1)
				}
				CDRRecCount.IncN(thread.Name, CDR)
				CDRPerSec.Store(thread.Name, 0)
			}
		case <-heartbeat10:
			log.Println("INFO: 10 second generator statistics")
			for _, thread := range global_cfg.Tasks {
				log.Println("INFO: Load " + strconv.Itoa(CDRRecCount.Load(thread.Name)) + " records for " + thread.Name)
				if tofile {
					log.Println("INFO: Save " + strconv.Itoa(CDRFileCount.Load(thread.Name)) + " files for " + thread.Name)
				}
				for _, t := range thread.RecTypeRatio {
					log.Println("INFO: Load " + thread.Name + " calls type " + t.Record_type + " type service " + t.TypeService + "(" + t.Name + ") " + CDRRecTypeCount.LoadString(thread.Name, t.Name))
				}
			}
		}
	}
}

// Горутина формирования CDR для файла
func StartTaskFile(PoolList []data.RecTypePool, cfg data.TasksType, FirstStart bool) {

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
		// при БРТ не стартует поток ВРЕМЕННО!!!
		if FirstStart && !brt {
			if Flag.Load(cfg.Name) == 1 {
				log.Println("INFO: Start new thead " + cfg.Name)
				go StartTaskFile(PoolList, cfg, false)
				Flag.Store(cfg.Name, 0)
			}
		}

		CDR = CDRPerSec.Load(cfg.Name)
		if CDR < cfg.CallsPerSecond {
			// Сброс счетчика
			if PoolIndex >= PoolIndexMax {
				PoolIndex = 0
			}

			PoolIndex++

			tmp = PoolList[PoolIndex].CallsCount

			if tmp != 0 {

				PoolList[PoolIndex].CallsCount = tmp - 1

				RecTypeIndex = data.RandomRecType(cfg.RecTypeRatio, rand.Intn(100))
				CDRRecTypeCount.Inc(cfg.Name, cfg.RecTypeRatio[RecTypeIndex].Name)

				//Запись в фаил
				// Формирование готовой строки для записи в фаил
				// Скорее всего роуминг пишем только файлы

				//для расчета постфактум
				//CDRPerSec.Inc(cfg.Name)
				rr, err := data.CreateCDRRecord(PoolList[PoolIndex], time.Now(), cfg.RecTypeRatio[RecTypeIndex], cfg.CDR_pattern)

				CDRChanneltoFileUni[cfg.Name] <- rr

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
	ProcessDebug("Start write " + f.Name())

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

// Горутина формирования данных для вызова Diameter
func StartTaskDiam(PoolList []data.RecTypePool, cfg data.TasksType, FirstStart bool) {
	{

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
			// при БРТ не стартует поток ВРЕМЕННО!!!
			if FirstStart && !brt {
				if Flag.Load(cfg.Name) == 1 {
					log.Println("INFO: Start new thead " + cfg.Name)
					go StartTaskDiam(PoolList, cfg, false)
					Flag.Store(cfg.Name, 0)
				}
			}

			CDR = CDRPerSec.Load(cfg.Name)
			if CDR < cfg.CallsPerSecond {
				// Сброс счетчика
				if PoolIndex >= PoolIndexMax {
					PoolIndex = 0
				}

				PoolIndex++

				tmp = PoolList[PoolIndex].CallsCount

				if tmp != 0 {

					PoolList[PoolIndex].CallsCount = tmp - 1

					RecTypeIndex = data.RandomRecType(cfg.RecTypeRatio, rand.Intn(100))
					CDRRecTypeCount.Inc(cfg.Name, cfg.RecTypeRatio[RecTypeIndex].Name)

					diam_message, err := data.CreateCCREventMessage(PoolList[PoolIndex], time.Now(), cfg.RecTypeRatio[RecTypeIndex], dict.Default)
					BrtDiamChannel <- data.DiamCH{TaskName: cfg.Name, Message: diam_message}
					// Что бы не завалить на время тестов
					time.Sleep(7 * time.Second)
					//Для БРТ считаем здесь. Пока здесь, похоже фаил пишется дольше
					CDRPerSec.Inc(cfg.Name)

					if err != nil {
						ErrorChannel <- err
					}
				}
			}
		}
	}
}

// Запуск потоков подключения к БРТ
func StartDiameterClient() {

	var brt_adress []datatype.Address
	for _, ip := range global_cfg.Common.BRT {
		brt_adress = append(brt_adress, datatype.Address(net.ParseIP(ip)))
	}

	//прописываем конфиг
	ProcessDebug("Load Diameter config")

	diam_cfg := &sm.Settings{
		OriginHost:       datatype.DiameterIdentity(global_cfg.Common.BRT_OriginHost),
		OriginRealm:      datatype.DiameterIdentity(global_cfg.Common.BRT_OriginRealm),
		VendorID:         data.PETER_SERVICE_VENDOR_ID,
		ProductName:      "CDR-generator",
		OriginStateID:    datatype.Unsigned32(time.Now().Unix()),
		FirmwareRevision: 1,
		HostIPAddresses:  brt_adress,
	}

	// Create the state machine (it's a diam.ServeMux) and client.
	mux := sm.New(diam_cfg)
	ProcessDebug(mux.Settings())

	ProcessDebug("Load Diameter dictionary")
	ProcessDebug("Load Diameter client")

	cli := &sm.Client{
		Dict:               data.Default, //dict.Default,
		Handler:            mux,
		MaxRetransmits:     3,
		RetransmitInterval: time.Second,
		EnableWatchdog:     true,
		WatchdogInterval:   5 * time.Second,

		AuthApplicationID: []*diam.AVP{
			//AVP Auth-Application-Id (код 258) имеет тип Unsigned32 и используется для публикации поддержки  Authentication and Authorization части diameter приложения (см. Section 2.4).
			//Если AVP Auth-Application-Id присутствует в сообщении, отличном от CER и CEA, значение этого AVP ДОЛЖНО соответствовать Application-Id, присутствующему в заголовке этого сообщения Diameter.
			diam.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(4)), // RFC 4006
		},
		AcctApplicationID: []*diam.AVP{
			// Acct-Application-Id AVP (AVP-код 259) имеет тип Unsigned32 и используется для публикации поддержки  Accountingand части diameter приложения (см. Section 2.4)
			//Если AVP Acct-Application-Id присутствует в сообщении, отличном от CER и CEA, значение этого AVP ДОЛЖНО соответствовать Application-Id, присутствующему в заголовке этого сообщения Diameter.
			diam.NewAVP(avp.AcctApplicationID, avp.Mbit, 0, datatype.Unsigned32(3)), //3
		},

		//Vendor-Specific-Application-Id AVP
		//Vendor-Specific-Application-Id AVP (код 260) имеет тип Grouped и используется для публикации поддержки vendor-specific Diameter-приложения. Точно один экземпляр Auth-Application-Id или Acct-Application-Id AVP ДОЛЖЕН присутствовать в составе этого AVP. Идентификатор приложения, переносимый либо Auth-Application-Id, либо Acct-Application-Id AVP, ДОЛЖЕН соответствовать идентификатору приложения конкретного поставщика, описанному в (Section 11.3 наверное 5.3). Он ДОЛЖЕН также соответствовать идентификатору приложения, присутствующему в заголовке Diameter сообщений, за исключением  сообщении CER или CEA.
		//
		//AVP Vendor-Id - это информационный AVP, относящийся к поставщику, который может иметь авторство конкретного приложения Diameter. Он НЕ ДОЛЖЕН использоваться в качестве средства определения отдельного пространства идентификаторов Application-Id.
		//
		//AVP Vendor-Specific-Application-Id  ДОЛЖЕН быть установлен как можно ближе к заголовку Diameter.
		//
		//     AVP Format
		//      <Vendor-Specific-Application-Id> ::= < AVP Header: 260 >
		//                                           { Vendor-Id }
		//                                           [ Auth-Application-Id ]
		//                                          [ Acct-Application-Id ]
		//AVP Vendor-Specific-Application-Id  ДОЛЖЕН содержать только один из идентификаторов Auth-Application-Id или Acct-Application-Id. Если AVP Vendor-Specific-Application-Id получен без одного из этих двух AVP, то получатель ДОЛЖЕН вернуть ответ с Result-Code DIAMETER_MISSING_AVP. В ответ СЛЕДУЕТ также включить Failed-AVP, который ДОЛЖЕН содержать пример AVP Auth-Application-Id и AVP Acct-Application-Id.
		//
		//Если получен AVP Vendor-Specific-Application-Id, содержащий оба идентификатора Auth-Application-Id и Acct-Application-Id, то получатель ДОЛЖЕН выдать ответ с Result-Code DIAMETER_AVP_OCCURS_TOO_MANY_TIMES. В ответ СЛЕДУЕТ также включить два Failed-AVP, которые содержат полученные AVP Auth-Application-Id и Acct-Application-Id.
		VendorSpecificApplicationID: []*diam.AVP{
			diam.NewAVP(avp.VendorSpecificApplicationID, avp.Mbit, 0, &diam.GroupedAVP{
				AVP: []*diam.AVP{
					diam.NewAVP(avp.VendorID, avp.Mbit, 0, datatype.Unsigned32(data.PETER_SERVICE_VENDOR_ID)),
					diam.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(4)),
					//diam.NewAVP(avp.AcctApplicationID, avp.Mbit, 0, datatype.Unsigned32(4)),
				},
			}),
		},
		SupportedVendorID: []*diam.AVP{
			diam.NewAVP(avp.VendorSpecificApplicationID, avp.Mbit, 0, &diam.GroupedAVP{
				AVP: []*diam.AVP{
					diam.NewAVP(avp.VendorID, avp.Mbit, 0, datatype.Unsigned32(data.PETER_SERVICE_VENDOR_ID)),
					diam.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(4)),
				},
			}),
		},
	}

	// Set message handlers.
	// Можно использовать канал AnswerCCAEvent(BrtDiamChannelAnswer)
	mux.Handle("CCA", AnswerCCAEvent())
	mux.Handle("DWA", AnswerDWAEvent())

	// Запуск потока записи ошибок в лог
	go DiamPrintErrors(mux.ErrorReports())
	//KeepAlive WTF??
	cli.EnableWatchdog = false //true

	brt_connect := make([]diam.Conn, len(brt_adress))

	var i int
	i = 0

	log.Println("DIAM: Connecting clients...")
	for _, init_connect := range global_cfg.Common.BRT {
		ProcessDebug(init_connect)

		var err error

		brt_connect[i], err = Dial(cli, init_connect+":"+strconv.Itoa(global_cfg.Common.BRT_port), "", "", false, "tcp")
		if err != nil {
			log.Println("Connect error ")
			log.Fatal(err)
		}

		ProcessDebug("Connect to " + init_connect + " done.")
		// Запуск потоков записи
		go SendCCREvent(brt_connect[i], diam_cfg, BrtDiamChannel)
		i++
	}

	log.Println("DIAM: Done. Sending messages...")

}

// Кусок для диаметра
// Определение шифрование соединения
func Dial(cli *sm.Client, addr, cert, key string, ssl bool, networkType string) (diam.Conn, error) {
	if ssl {
		return cli.DialNetworkTLS(networkType, addr, cert, key, nil)
	}
	return cli.DialNetwork(networkType, addr)
}

//Обработчик-ответа Диаметра
func AnswerCCAEvent() diam.HandlerFunc {
	//func AnswerCCAEvent(done chan struct{}) diam.HandlerFunc {
	return func(c diam.Conn, m *diam.Message) {
		//обработчик ошибок
		data.ResponseDiamHandler(m, log.Default(), debugm)
		log.Println(m)
	}
}

func AnswerDWAEvent() diam.HandlerFunc {
	return func(c diam.Conn, m *diam.Message) {
		//обработчик ошибок
		data.ResponseDiamHandler(m, log.Default(), debugm)
	}
}

// Горутина приема и записи сообщения по диаметру в брт
func SendCCREvent(c diam.Conn, cfg *sm.Settings, in chan data.DiamCH) {

	var err error
	server, _, _ := strings.Cut(c.RemoteAddr().String(), ":")
	// на подумать, использовать структуру, а потом ее определять или сазу передавать готовое сообщение
	// заменить на просто вывод в лог
	defer c.Close()

	heartbeat := time.Tick(5 * time.Second)
	meta, ok := smpeer.FromContext(c.Context())
	if !ok {
		log.Println("Client connection does not contain metadata")
		log.Println("Close threads")
	}

	for {
		select {
		case <-heartbeat:
			// Сделать выход или переоткрытие?
			meta, ok = smpeer.FromContext(c.Context())
			if !ok {
				log.Println("Client connection does not contain metadata")
				log.Println("Close threads")
			}

			// Настройка Watch Dog
			m := diam.NewRequest(280, 4, dict.Default)
			m.NewAVP(avp.OriginHost, avp.Mbit, 0, cfg.OriginHost)
			m.NewAVP(avp.OriginRealm, avp.Mbit, 0, cfg.OriginRealm)
			m.NewAVP(avp.OriginStateID, avp.Mbit, 0, cfg.OriginStateID)
			log.Printf("DIAM: Sending DWR to %s", c.RemoteAddr())
			_, err = m.WriteTo(c)
			if err != nil {
				ErrorChannel <- err
			}

		case tmp := <-in:

			diam_message := tmp.Message
			//diam_message := data.CreateCCREventMessage(dict.Default)
			diam_message.NewAVP(avp.OriginHost, avp.Mbit, 0, cfg.OriginHost)
			diam_message.NewAVP(avp.OriginRealm, avp.Mbit, 0, cfg.OriginRealm)
			diam_message.NewAVP(avp.DestinationRealm, avp.Mbit, 0, meta.OriginRealm)
			diam_message.NewAVP(avp.DestinationHost, avp.Mbit, 0, meta.OriginHost)

			//log.Println("DIAM: Sending to ", c.RemoteAddr())
			log.Printf("DIAM: Sending CCR to %s", c.RemoteAddr())

			_, err = diam_message.WriteTo(c)
			if err != nil {
				ErrorChannel <- err
			} else {
				//CDRPerSec.Inc(tmp.TaskName)
				CDRBRTCount.Inc(server)
			}

		default:

		}
	}

}

// Поток телнета Кемел
// два типа каналов CDR и закрытие/переоткрытие
// +надо сделать keepalive
func StartCamelTelnet() {

}
