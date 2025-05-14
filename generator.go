package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/signal"
	"sync"
	"syscall"

	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"context"

	"github.com/egorkovalchuk/go-cdrgenerator/pkg/data"
	"github.com/egorkovalchuk/go-cdrgenerator/pkg/diameter"
	"github.com/egorkovalchuk/go-cdrgenerator/pkg/influx"
	"github.com/egorkovalchuk/go-cdrgenerator/pkg/pid"
	"github.com/egorkovalchuk/go-cdrgenerator/pkg/tlv"
	"github.com/fiorix/go-diameter/v4/diam"
	"github.com/fiorix/go-diameter/v4/diam/avp"
	"github.com/fiorix/go-diameter/v4/diam/datatype"
	"github.com/fiorix/go-diameter/v4/diam/dict"
	"github.com/fiorix/go-diameter/v4/diam/sm"
	"github.com/fiorix/go-diameter/v4/diam/sm/smpeer"
	//"github.com/pkg/profile"
)

//Power by  Egor Kovalchuk

const (
	logFileName = "generator.log"
	pidFileName = "generator.pid"
	versionutil = "0.5.6"
)

var (
	// конфиг
	global_cfg data.Config
	// режим работы сервиса(дебаг мод)
	debugm bool
	// ошибки
	err error
	// режим работы сервиса
	startdaemon bool
	// запрос версии
	version bool
	// Запись в фаил
	tofile bool
	// для выбора типа соединения
	brt     bool
	camel   bool
	brtlist data.ArgListType

	// Каналы для управления и передачи информации
	ProcessChannel = make(chan string)
	LogChannel     = make(chan LogStruct, 1000)

	// Признак запуска дополнительного потока
	Flag = data.NewFlag()
	// Скорость потока
	CDRPerSec = data.NewCounters()
	// Скорость потока Camel
	CDRPerSecCamel = data.NewCounters()
	// Скорость потока Diameter
	CDRPerSecDiam = data.NewCounters()
	// Срез для каналов для записи файлов
	CDRChanneltoFileUni = make(map[string](chan string))
	// Срез CDR_Pattern, для уменьшения использовании памяти вынесена из массива
	CDRPatternTask = make(map[string]data.CDRPatternType)

	// Статистика записи
	CDRRecCount           = data.NewCounters()
	CDRFileCount          = data.NewCounters()
	CDRRecTypeCount       = data.NewRecTypeCounters()
	CDRDiamCount          = data.NewCounters()
	CDRDiamRCount         = data.NewCounters()
	CDRDiamResponseCount  = data.NewCounters()
	CDRCamelCount         = data.NewCounters()
	CDRCamelRCount        = data.NewCounters()
	CDRCamelResponseCount = data.NewCounters()

	// Канал для Диметра коннект к БРТ и Camel
	BrtDiamChannelAnswer = make(chan diam.Message, 4000)
	BrtDiamChannel       = make(chan diameter.DiamCH, 4000)
	BrtOfflineCDR        = data.NewCDROffline()
	CamelOfflineCDR      = data.NewCDROffline()

	// Канал записи в Camel
	CamelChannel  = make(chan tlv.Camel_tcp, 4000)
	list_listener *tlv.ListListener
	camelserver   *tlv.Server
	WriteChan     = make(chan tlv.WriteStruck, 4000)

	// Канал записи статистики в БД
	ReportStat = make(chan string, 1000)

	// Срез для LAC/CELL
	LACCELLpool = make(map[string]([]data.RecTypeLACPool))
	LACCELLlen  = make(map[string](int))

	// Маркер завершения горутин генерации нагрузки
	gostop = false
	// Тестовый параметр замедления
	slow       bool // Равномерная запись
	slow_camel bool // Запись раз в 10 секунд - можно удали что бы не съедать машинное время

	// Удаление файлов после работы демона
	// Тестовая опция для удобства
	rm bool
	// Разрешить запускать дочернии процессы
	thread_secodary bool

	wg sync.WaitGroup

	// контексты
	ctx  context.Context
	stop context.CancelFunc
)

func main() {
	//start program
	var argument string

	if len(os.Args) > 1 {
		argument = os.Args[1]
	} else {
		data.HelpStart()
		return
	}

	if argument == "-h" {
		data.HelpStart()
		return
	} else if argument == "-s" {
		err = pid.StopProcess(pidFileName)
		if err != nil {
			fmt.Println(err.Error())
		}
		return
	}

	flag.BoolVar(&debugm, "debug", false, "Start with debug mode")
	flag.BoolVar(&startdaemon, "d", false, "Start SCP server")
	flag.BoolVar(&version, "v", false, "Print version")
	flag.BoolVar(&brt, "brt", false, "Connect to BRT Diameter protocol")
	flag.BoolVar(&tofile, "file", false, "Start save CDR to file")
	flag.BoolVar(&camel, "camel", false, "SCP(Camel) Server for BRT (Camel protocol)")
	flag.Var(&brtlist, "brtlist", "List of name task to work BRT")
	flag.BoolVar(&rm, "rm", false, "Delete files")
	flag.BoolVar(&thread_secodary, "thread", false, "Enable start new threads")
	// замедление и тесты
	flag.BoolVar(&slow, "slow", false, "Start with slow mode")
	flag.BoolVar(&slow_camel, "slow_camel", false, "Start with slow_camel mode")
	flag.Parse()

	// Открытие лог файла
	// ротация не поддерживается в текущей версии
	// Вынести в горутину
	filer, err := os.OpenFile(logFileName, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer filer.Close()

	log.SetOutput(filer)

	// запуск горутины записи в лог
	go LogWriteForGoRutineStruct(LogChannel)

	ProcessInfo("Start util")
	ProcessDebug("Start with debug mode")

	if startdaemon {
		ProcessInfo("Start daemon mode")
		fmt.Println("Start util in daemon mode")
	}

	if version {
		fmt.Println("Version utils " + versionutil)
		return
	}

	// Чтение конфига
	global_cfg.ReadConf("config.json")

	// инициализация переменных
	InitVariables()

	// создаем pid
	err = pid.SetPID(pidFileName)
	if err != nil {
		ProcessError("Can not create pid-file: " + err.Error())
	}

	// запуск контекста
	ctx = context.Background()
	ctx, stop = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Запуск мониторинга
	// вркменно отключено для демона
	// сделать задержку запуска мониторинга. запускается ранбше чем статистика начинает писаться.
	go Monitor()
	// Запуск статистики
	// как переопределить монитор?
	if global_cfg.Common.Report.Influx {

		w := influx.NewInfluxWriter(&influx.Config{
			InfluxToken:   global_cfg.Common.Report.InfluxToken,
			InfluxOrg:     global_cfg.Common.Report.InfluxOrg,
			InfluxVersion: global_cfg.Common.Report.InfluxVersion,
			InfluxBucket:  global_cfg.Common.Report.InfluxBucket,
			InfluxServer:  global_cfg.Common.Report.InfluxServer,
		}, ProcessInflux)
		// Запуск горутины записи в инфлюкс
		w.StartHTTPWriter(ReportStat)
	}

	// Определяем ветку
	// Запускать горутиной с ожиданием процесса сигналов от ОС
	if startdaemon {
		StartDaemonMode()
		ProcessInfo("Daemon terminated")
	} else {
		StartSimpleMode()
	}
	fmt.Println("Done")
}

// StartSimpleMode запуск в режиме скрипта
func StartSimpleMode() {
	// запускаем отдельные потоки родительские потоки для задач из конфига
	// родительские породождают дочерние по формуле
	// при завершении времени останавливают дочерние и сами
	// Добавить сигнал остановки

	// Запуски Диаметр и Кемел
	StartClient()

	// Основной цикл
	for i, thread := range global_cfg.Tasks {
		if thread.DatapoolCsvFile == "" {
			ProcessWarm("Please, set the file name specified for " + thread.Name)
		} else {
			f, err := os.Open(thread.DatapoolCsvFile)
			if err != nil {
				ProcessErrorAny("Unable to read input file "+thread.DatapoolCsvFile, err)
				ProcessError("Thread " + thread.Name + " not start")
			} else {
				defer f.Close()
				ProcessInfo("Start load pool " + thread.DatapoolCsvFile)

				// read csv values using csv.Reader
				csvReader := csv.NewReader(f)
				// Разделитель CSV
				csvReader.Comma = ';'
				csv, err := csvReader.ReadAll()

				// Заполнение количества звонков на абонента
				var PoolList data.PoolSubs
				PoolList = PoolList.CreatePoolList(csv, thread)

				if err != nil {
					ProcessError(err)
				} else if len(PoolList) == 0 {
					ProcessError("Pool empty. Skeep")
				} else {
					ProcessDebug("Last record ")
					ProcessDebug(PoolList[len(PoolList)-1])

					ProcessInfo("Load " + strconv.Itoa(len(PoolList)) + " records")

					// Включаем вывод статистики
					global_cfg.Tasks[i].Pool_loading = true

					if camel {
						go StartTaskCamel(PoolList, thread, true)
						// Запускаем запись для офлайна - ответ брт != 0 или параметр из конфига
						go StartFileCDR(thread, CDRChanneltoFileUni[thread.Name])
					} else if brt && brtlist.Get(thread.Name) {
						ProcessInfo("Start diameter thread for " + thread.Name)
						// Запуск задачи формирования вызовов и отправки в БРТ по диаметру
						go StartTaskDiam(PoolList, thread, true)
						// Запускаем запись для офлайна - ответ брт 4011
						go StartFileCDR(thread, CDRChanneltoFileUni[thread.Name])
					} else if tofile {
						// Если пишем в фаил, в начале запускаем потоки записи в фаил или коннекта к BRT
						// Запускать потоки записи по количеству путей? Не будет ли пересечение по именам файлов
						ProcessInfo("Start file thread for " + thread.Name)
						go StartFileCDR(thread, CDRChanneltoFileUni[thread.Name])
						go StartTaskFile(PoolList, thread, true)
					} else {
						ProcessInfo("Usage options not set " + thread.Name)
					}

				}

			}
		}
	}
	end()
}

// Функция контроля рейтов
// Подсчетов типов отправленных сообщений
func Monitor() {
	heartbeat := time.Tick(1 * time.Second)
	heartbeat10 := time.Tick(10 * time.Second)

	var CDR int
	var CDRCamel int
	var CDRDiam int
	var checkstart int

	tmpCamel := make(map[string]int)
	tmpDiam := make(map[string]int)

	time.Sleep(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			ProcessInfo("Stop monitor")
			return
		case <-heartbeat:
			for _, thread := range global_cfg.Tasks {
				if thread.Pool_loading {
					CDR = CDRPerSec.Load(thread.Name)
					ProcessDebug("Speed task " + thread.Name + " " + strconv.Itoa(CDR) + " op/s")
					if camel {
						CDRCamel = CDRPerSecCamel.Load(thread.Name)
						ProcessDebug("Speed camel task " + thread.Name + " " + strconv.Itoa(CDRCamel) + " op/s")
					}
					if brt && brtlist.Get(thread.Name) {
						CDRDiam = CDRPerSecDiam.Load(thread.Name)
						ProcessDebug("Speed Diameter task " + thread.Name + " " + strconv.Itoa(CDRDiam) + " op/s")
					}
					// вынести отдельно? что бы уменьшить выполнения условий
					if global_cfg.Common.Report.Influx {
						ReportStat <- "cdr_offline,region=" + global_cfg.Common.Report.Region + ",task_name=" + thread.Name + " speed=" + strconv.Itoa(CDR)
						if camel {
							ReportStat <- "cdr_camel,region=" + global_cfg.Common.Report.Region + ",task_name=" + thread.Name + " speed=" + strconv.Itoa(CDRCamel)
						}
						if brt && brtlist.Get(thread.Name) {
							ReportStat <- "cdr_brt,region=" + global_cfg.Common.Report.Region + ",task_name=" + thread.Name + " speed=" + strconv.Itoa(CDRDiam)
						}
					}

					if CDR < thread.CallsPerSecond {
						// запускаем еще один поток если только 10 секунд подряд скорость была ниже
						if checkstart > 10 {
							Flag.Store(thread.Name, 1)
							checkstart = 0
						}
						checkstart++
					} else {
						checkstart = 0
					}
					CDRRecCount.IncN(thread.Name, CDR)
					CDRPerSec.Store(thread.Name, 0)
					CDRPerSecCamel.Store(thread.Name, 0)
					CDRPerSecDiam.Store(thread.Name, 0)
				}
			}
		case <-heartbeat10:
			ProcessInfo("10 second generator statistics")
			for _, thread := range global_cfg.Tasks {
				ProcessInfo("Load " + strconv.Itoa(CDRRecCount.Load(thread.Name)) + " records for " + thread.Name)
				if tofile {
					ProcessInfo("Save " + strconv.Itoa(CDRFileCount.Load(thread.Name)) + " files for " + thread.Name)
				}
				for _, t := range thread.RecTypeRatio {
					ProcessInfo("Load " + thread.Name + " calls type " + t.Record_type + " type service " + t.TypeService + "(" + t.Name + ") " + CDRRecTypeCount.LoadString(thread.Name, t.Name))
				}
			}
			if brt {
				for _, ip := range global_cfg.Common.BRT {
					ProcessInfo("Send " + strconv.Itoa(CDRDiamCount.Load(ip)) + " Diameter messages to " + ip)
				}
				CDRDiamResponseCount.LoadRangeToLogFunc("Diameter response code ", ProcessInfo)
				if global_cfg.Common.Report.Influx {
					tmpDiam = CDRDiamResponseCount.LoadMapSpeed(tmpDiam, "brt", global_cfg.Common.Report.Region, ReportStat, ProcessInflux)
				}
			}
			if camel {
				CDRCamelCount.LoadRangeToLogFunc("Camel messages id  ", ProcessInfo)
				CDRCamelResponseCount.LoadRangeToLogFunc("Camel response code Brt id ", ProcessInfo)
				CDRCamelRCount.LoadRangeToLogFunc("Camel messages revc ", ProcessInfo)
				if global_cfg.Common.Report.Influx {
					tmpCamel = CDRCamelResponseCount.LoadMapSpeed(tmpCamel, "camel", global_cfg.Common.Report.Region, ReportStat, ProcessInflux)
				}
			}
		}
	}
}

// Горутина формирования CDR для файла
func StartTaskFile(PoolList data.PoolSubs, cfg data.TasksType, FirstStart bool) {

	var PoolIndex int
	var PoolIndexMax int
	var CDR int
	var RecTypeIndex int
	var tmp int
	var skeep int
	PoolIndex = 0
	PoolIndexMax = len(PoolList) - 1

	for {
		// Порождать доп процессы может только первый процесс
		// Уменьшает обращение с блокировками переменной флаг
		if FirstStart && thread_secodary {
			if Flag.Load(cfg.Name) == 1 {
				ProcessInfo("Start new thead " + cfg.Name)
				go StartTaskFile(PoolList, cfg, false)
				Flag.Store(cfg.Name, 0)
			}
		}

		CDR = CDRPerSec.Load(cfg.Name)
		if CDR < cfg.CallsPerSecond {
			// Сброс счетчика
			if PoolIndex >= PoolIndexMax {
				PoolIndex = 0
				skeep = 0
			}

			PoolIndex++
			tmp = PoolList[PoolIndex].CallsCount
			if tmp != 0 {

				PoolList[PoolIndex].CallsCount = tmp - 1

				RecTypeIndex = data.RandomRecType(cfg.RecTypeRatio, rand.Intn(100))
				CDRRecTypeCount.Inc(cfg.Name, cfg.RecTypeRatio[RecTypeIndex].Name)

				// Запись в фаил
				// Формирование готовой строки для записи в фаил
				// Скорее всего роуминг пишем только файлы
				lc := LACCELLpool[cfg.Name][rand.Intn(LACCELLlen[cfg.Name])]
				rr, err := data.CreateCDRRecord(PoolList[PoolIndex], time.Now(), cfg.RecTypeRatio[RecTypeIndex], CDRPatternTask[cfg.Name], data.RandomMSISDN(cfg.Name), lc)
				if err != nil {
					ProcessError(err)
				} else {
					CDRChanneltoFileUni[cfg.Name] <- rr
				}
			} else {
				skeep++
				if skeep == PoolIndexMax {
					ProcessInfo("Re-redaing pool")
					PoolList.ReinitializationPoolList(cfg)
				}
			}
		}
	}
}

// Запись в Фаил
// Исправил на канал для чтения
func StartFileCDR(task data.TasksType, InputString chan string) {
	// Запись в разные каталоги
	var DirNum int
	DirNumLen := len(task.PathsToSave)

	f, err := os.Create(strings.Replace(task.PathsToSave[0]+task.Template_save_file, "{date}", time.Now().Format("20060201030405"), 1))

	if err != nil {
		ProcessError(err)
		if os.IsPermission(err) {
			gostop = true
			stop()
			return
		}
	}

	ProcessDebug("Start write " + f.Name())

	defer f.Close()

	// Переcoздание файла каждве 2 секунды
	heartbeat := time.Tick(2 * time.Second)

	for {
		select {
		case <-heartbeat:
			f.Close()
			// Директория меняется рандомно
			DirNum = rand.Intn(DirNumLen)
			f, err = os.Create(strings.Replace(task.PathsToSave[DirNum]+task.Template_save_file, "{date}", time.Now().Format("20060201030405"), 1))

			if err != nil {
				ProcessError(err)
				if os.IsPermission(err) {
					gostop = true
					stop()
					return
				}
			}

			// Переместить в горутину
			ProcessDebug("Start write " + f.Name())

			CDRFileCount.Inc(task.Name)
			defer f.Close() //Закрыть фаил при нешаттном завершении
		default:

			str := <-InputString
			// Перенес из генерации, в одном потоке будет работать быстрее
			// Исключаем запись из брт и Кемел
			if brtlist.Get(task.Name) || camel {
				CDRRecCount.Inc(task.Name + "offline")
			} else {
				CDRPerSec.Inc(task.Name)
			}

			_, err = f.WriteString(str + "\n")

			if err != nil {
				ProcessError(err)
			}
		}
	}
}

// Чтение из потока файлов (работа без генератора)
func StartTransferCDR(FileName string, InputString <-chan string) {

}

// Горутина формирования данных для вызова Diameter
func StartTaskDiam(PoolList data.PoolSubs, cfg data.TasksType, FirstStart bool) {
	{
		var PoolIndex int
		var PoolIndexMax int
		var CDR int
		var RecTypeIndex int
		var tmp int
		var skeep int
		PoolIndex = 0
		PoolIndexMax = len(PoolList) - 1

		for {
			// Порождать доп процессы может только первый процесс
			// Уменьшает обращение с блокировками переменной флаг
			// при БРТ не стартует поток ВРЕМЕННО!!!
			// Поток подключение по диаметру 1, боль 10к/с разогнать
			// по одному подключению не получается
			if FirstStart && thread_secodary {
				if Flag.Load(cfg.Name) == 1 {
					ProcessInfo("Start new thead " + cfg.Name)
					go StartTaskDiam(PoolList, cfg, false)
					Flag.Store(cfg.Name, 0)
				}
			}
			// Выход из горутины
			if gostop {
				return
			}

			CDR = CDRPerSec.Load(cfg.Name)
			if CDR < cfg.CallsPerSecond {
				// Сброс счетчика
				if PoolIndex >= PoolIndexMax {
					PoolIndex = 0
					skeep = 0
				}

				PoolIndex++
				tmp = PoolList[PoolIndex].CallsCount
				if tmp != 0 {

					PoolList[PoolIndex].CallsCount = tmp - 1
					RecTypeIndex = data.RandomRecType(cfg.RecTypeRatio, rand.Intn(100))
					CDRRecTypeCount.Inc(cfg.Name, cfg.RecTypeRatio[RecTypeIndex].Name)

					// Стараемся отсылать равномерно, в условии что, пропускать уже обнуленные CallsCount
					if slow {
						sleep(time.Duration(cfg.Time_delay) * time.Nanosecond)
					}
					CreateDiamMessage(PoolList[PoolIndex], cfg.Name, cfg.RecTypeRatio[RecTypeIndex])
				} else {
					skeep++
					if skeep == PoolIndexMax {
						ProcessInfo("Re-redaing pool")
						PoolList.ReinitializationPoolList(cfg)
					}
				}
			}
		}
	}
}

// Запуск потоков подключения к БРТ
func StartDiameterClient() {

	var brt_adress []datatype.Address
	localip := data.GetLocalIP()
	for _, ip := range global_cfg.Common.BRT {
		if ip == localip {
			brt_adress = append(brt_adress, datatype.Address(net.ParseIP("127.0.0.1")))
		}
		brt_adress = append(brt_adress, datatype.Address(net.ParseIP(ip)))
	}
	brt_adress = append(brt_adress, datatype.Address(net.ParseIP(localip)))
	// счетчик активных соединений
	chk := 0

	//прописываем конфиг
	ProcessDebug("Load Diameter config")

	diam_cfg := &sm.Settings{
		OriginHost:       datatype.DiameterIdentity(global_cfg.Common.BRT_OriginHost),
		OriginRealm:      datatype.DiameterIdentity(global_cfg.Common.BRT_OriginRealm),
		VendorID:         diameter.PETER_SERVICE_VENDOR_ID,
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

	// Инициализация конфига клиента
	cli := diameter.Client(mux)

	// Set message handlers.
	// Можно использовать канал AnswerCCAEvent(BrtDiamChannelAnswer)
	mux.Handle("CCA", AnswerCCAEvent())
	mux.Handle("DWA", AnswerDWAEvent())
	mux.Handle("ALL", AnswerALLEvent())
	//go DiamAnswer(BrtDiamChannelAnswer)

	// Запуск потока записи ошибок в лог
	go DiamPrintErrors(mux.ErrorReports())

	ProcessDiam("Connecting clients...")
	for _, init_connect := range global_cfg.Common.BRT {

		ProcessDebug(init_connect)
		var err error

		brt_connect, err := Dial(cli, init_connect+":"+strconv.Itoa(global_cfg.Common.BRT_port), "", "", false, "tcp")

		if err != nil {
			ProcessError("Connect error ")
			ProcessError(err)
		} else {
			ProcessDebug("Connect to " + init_connect + " done.")
			// Запуск потоков записи по БРТ
			// Отмеаем что клиент запущен
			chk++
			go SendCCREvent(brt_connect, diam_cfg, BrtDiamChannel)
		}
	}
	// Проверка что клиент запущен
	if chk > 0 {
		ProcessDiam("Done. Sending messages...")
	} else {
		ProcessDiam("Stopping the client's diameter. No connection is initialized")
		brt = false
	}
}

// Кусок для диаметра
// Определение шифрование соединения
func Dial(cli *sm.Client, addr, cert, key string, ssl bool, networkType string) (diam.Conn, error) {
	if ssl {
		return cli.DialNetworkTLS(networkType, addr, cert, key, nil)
	}
	return cli.DialNetwork(networkType, addr)
}

// Тест горутина обработки ответов диаметра
func DiamAnswer(f chan diam.Message) {
	for m := range f {
		s, sid := diameter.ResponseDiamHandler(&m, ProcessDiam, debugm)
		CDRDiamResponseCount.Inc(strconv.Itoa(s))
		if s == 4011 || s == 4522 || s == 4012 {
			//logdiam.Println("DIAM: Answer CCA code: " + strconv.Itoa(s) + " Session: " + sid)
			//переход в оффлайн
			val := BrtOfflineCDR.Load(sid) //BrtOfflineCDR[sid]*
			rr, err := data.CreateCDRRecord(val.RecPool, val.CDRtime, val.Ratio, CDRPatternTask[val.TaskName], val.DstMsisdn, data.RecTypeLACPool{LAC: val.Lac, CELL: val.Cell})
			if err != nil {
				ProcessError(err)
			} else {
				CDRChanneltoFileUni[val.TaskName] <- rr
			}
			BrtOfflineCDR.Delete(sid)
		} else if s == 5030 {
			// 5030 пользователь не известен
			BrtOfflineCDR.Delete(sid)
		} else {
			//logdiam.Println("DIAM: Answer CCA code: " + strconv.Itoa(s))
			BrtOfflineCDR.Delete(sid)
		}
	}
}

// Обработчик-ответа Диаметра
// f chan<- diam.Message
func AnswerCCAEvent() diam.HandlerFunc {
	return func(c diam.Conn, m *diam.Message) {
		go func() {
			// Конкуренция по ответам, запись в фаил?
			s, sid := diameter.ResponseDiamHandler(m, ProcessDiam, debugm)
			CDRDiamResponseCount.Inc(strconv.Itoa(s))
			CDRDiamRCount.Inc(c.RemoteAddr().String())
			if s == 4011 || s == 4522 || s == 4012 {
				//logdiam.Println("DIAM: Answer CCA code: " + strconv.Itoa(s) + " Session: " + sid)
				//переход в оффлайн
				val := BrtOfflineCDR.Load(sid) //BrtOfflineCDR[sid]*
				rr, err := data.CreateCDRRecord(val.RecPool, val.CDRtime, val.Ratio, CDRPatternTask[val.TaskName], val.DstMsisdn, data.RecTypeLACPool{LAC: val.Lac, CELL: val.Cell})
				if err != nil {
					ProcessError(err)
				} else {
					CDRChanneltoFileUni[val.TaskName] <- rr
				}
				BrtOfflineCDR.Delete(sid)
			} else if s == 5030 {
				// 5030 пользователь не известен
				BrtOfflineCDR.Delete(sid)
			} else {
				//logdiam.Println("DIAM: Answer CCA code: " + strconv.Itoa(s))
				BrtOfflineCDR.Delete(sid)
			}
		}()
	}
}

func AnswerDWAEvent() diam.HandlerFunc {
	return func(c diam.Conn, m *diam.Message) {
		//обработчик ошибок, вотч дог пишем в обычный лог
		s, _ := diameter.ResponseDiamHandler(m, ProcessDiam, debugm)
		ProcessDiam("Answer " + c.RemoteAddr().String() + " DWA code:" + strconv.Itoa(s))
	}
}

func AnswerALLEvent() diam.HandlerFunc {
	return func(c diam.Conn, m *diam.Message) {
		ProcessDiam(m)
	}
}

// Горутина  записи сообщения по диаметру в брт
func SendCCREvent(c diam.Conn, cfg *sm.Settings, in chan diameter.DiamCH) {

	var err error
	server, _, _ := strings.Cut(c.RemoteAddr().String(), ":")
	// на подумать, использовать структуру, а потом ее определять или сазу передавать готовое сообщение
	//defer c.Close()

	heartbeat := time.Tick(10 * time.Second)
	_, ok := smpeer.FromContext(c.Context())
	if !ok {
		ProcessDiam("Client connection does not contain metadata")
		ProcessDiam("Close threads")
	}

	for {
		select {
		case <-heartbeat:
			// Сделать выход или переоткрытие?
			_, ok := smpeer.FromContext(c.Context())
			if !ok {
				ProcessDiam("Client connection does not contain metadata")
				ProcessDiam("Close threads")

			}

			// Настройка Watch Dog
			m := diam.NewRequest(280, 4, dict.Default)
			m.NewAVP(avp.OriginHost, avp.Mbit, 0, cfg.OriginHost)
			m.NewAVP(avp.OriginRealm, avp.Mbit, 0, cfg.OriginRealm)
			m.NewAVP(avp.OriginStateID, avp.Mbit, 0, cfg.OriginStateID)
			ProcessDiam(fmt.Sprintf("Sending DWR to %s", c.RemoteAddr()))
			_, err = m.WriteTo(c)
			if err != nil {
				ProcessError(err)
			}

		case tmp := <-in:
			meta, ok := smpeer.FromContext(c.Context())
			if !ok {
				ProcessDiam("Client connection does not contain metadata")
				ProcessDiam("Close threads")
			}

			diam_message := tmp.Message
			diam_message.NewAVP(avp.OriginHost, avp.Mbit, 0, cfg.OriginHost)
			diam_message.NewAVP(avp.OriginRealm, avp.Mbit, 0, cfg.OriginRealm)
			diam_message.NewAVP(avp.DestinationRealm, avp.Mbit, 0, meta.OriginRealm)
			diam_message.NewAVP(avp.DestinationHost, avp.Mbit, 0, meta.OriginHost)

			_, err = diam_message.WriteTo(c)
			if err != nil {
				ProcessError(err)
			} else {
				CDRDiamCount.Inc(server)
			}
		}
	}
}

// StartDaemonMode запуск в режиме демона
func StartDaemonMode() {
	// запускаем отдельные потоки родительские потоки для задач из конфига
	// родительские породождают дочерние по формуле
	// при завершении времени останавливают дочерние и сами
	// Добавить сигнал остановки

	// Запуски Диаметр и Кемел
	StartClient()

	// Основной цикл
	for i, thread := range global_cfg.Tasks {
		if thread.DatapoolCsvFile == "" {
			ProcessWarm("Please, set the file name specified for " + thread.Name)
		} else {
			f, err := os.Open(thread.DatapoolCsvFile)
			if err != nil {
				ProcessErrorAny("Unable to read input file "+thread.DatapoolCsvFile, err)
				ProcessError("Thread " + thread.Name + " not start")
			} else {
				defer f.Close()
				ProcessInfo("Start load pool " + thread.DatapoolCsvFile)

				// read csv values using csv.Reader
				csvReader := csv.NewReader(f)
				// Разделитель CSV
				csvReader.Comma = ';'
				csv, err := csvReader.ReadAll()

				// Заполнение количества звонков на абонента
				var PoolList data.PoolSubs
				PoolList = PoolList.CreatePoolList(csv, thread)

				if err != nil {
					ProcessError(err)
				} else if len(PoolList) == 0 {
					ProcessError("Pool empty. Skeep")
				} else {
					ProcessDebug("Last record ")
					ProcessDebug(PoolList[len(PoolList)-1])

					ProcessInfo("Load " + strconv.Itoa(len(PoolList)) + " records")

					// Включаем вывод статистики
					global_cfg.Tasks[i].Pool_loading = true

					// Запис формируются в том числе для роуминга
					// Пока ограничено только локальными
					if camel {
						go StartTaskCamel(PoolList, thread, true)
						// Запускаем запись для офлайна - ответ брт != 0 или параметр из конфига
						go StartFileCDR(thread, CDRChanneltoFileUni[thread.Name])
					} else {
						ProcessInfo("Usage options not set " + thread.Name)
					}
				}
			}
		}
	}
	end()
}

// Горутина формирования данных для вызова Diameter
func StartTaskCamel(PoolList data.PoolSubs, cfg data.TasksType, FirstStart bool) {
	{
		var PoolIndex int
		var PoolIndexMax int
		var CDR int
		var RecTypeIndex int
		var tmp int
		var skeep int
		PoolIndex = 0
		PoolIndexMax = len(PoolList) - 1

		var ll bool
		if brt && brtlist.Get(cfg.Name) {
			ll = true
		}

		for {

			// Порождать доп процессы может только первый процесс
			// Уменьшает обращение с блокировками переменной флаг
			// Вынести в отдельный поток?
			if FirstStart && thread_secodary {
				if Flag.Load(cfg.Name) == 1 {
					ProcessInfo("Start new thead " + cfg.Name)
					go StartTaskCamel(PoolList, cfg, false)
					Flag.Store(cfg.Name, 0)
				}
			}
			// Выход из горутины
			if gostop {
				return
			}
			// пропуск шага если нет активных соединений
			// пропустить только для камел?
			if camel && len(list_listener.List) == 0 {
				ProcessInfo("Waiting to connect Camel client")
				time.Sleep(time.Duration(5) * time.Second)
				continue
			}

			CDR = CDRPerSec.Load(cfg.Name)
			if CDR < cfg.CallsPerSecond {
				// Сброс счетчика
				if PoolIndex >= PoolIndexMax {
					PoolIndex = 0
					skeep = 0
				}

				PoolIndex++
				tmp = PoolList[PoolIndex].CallsCount
				if tmp != 0 {

					PoolList[PoolIndex].CallsCount = tmp - 1
					RecTypeIndex = data.RandomRecType(cfg.RecTypeRatio, rand.Intn(100))
					CDRRecTypeCount.Inc(cfg.Name, cfg.RecTypeRatio[RecTypeIndex].Name)

					// Стараемся отсылать равномерно
					if slow {
						sleep(time.Duration(cfg.Time_delay) * time.Nanosecond)
					}

					switch {
					case cfg.RecTypeRatio[RecTypeIndex].DefaultChan == "camel":
						if slow_camel {
							time.Sleep(time.Duration(10) * time.Second)
						}
						CreateCamelMessage(PoolList[PoolIndex], cfg.Name, cfg.RecTypeRatio[RecTypeIndex])
					case cfg.RecTypeRatio[RecTypeIndex].DefaultChan == "diameter" && ll:
						CreateDiamMessage(PoolList[PoolIndex], cfg.Name, cfg.RecTypeRatio[RecTypeIndex])
					default:
						rr, err := data.CreateCDRRecord(PoolList[PoolIndex], time.Now(), cfg.RecTypeRatio[RecTypeIndex], CDRPatternTask[cfg.Name], data.RandomMSISDN(cfg.Name), LACCELLpool[cfg.Name][rand.Intn(LACCELLlen[cfg.Name])])
						if err != nil {
							ProcessError(err)
						} else {
							CDRChanneltoFileUni[cfg.Name] <- rr
							CDRPerSec.Inc(cfg.Name)
						}
					}
				} else {
					skeep++
					if skeep == PoolIndexMax {
						ProcessInfo("Re-redaing pool")
						PoolList.ReinitializationPoolList(cfg)
					}
				}
			}
		}
	}
}

// Старт потока Кемел
func StartCamelServer() {

	ProcessInfo("Start SCP Server")

	camel_cfg := &tlv.Config{
		Camel_port: global_cfg.Common.CAMEL.Port,
		//	Duration:         global_cfg.Common.Duration,
		Camel_SCP_id:     uint8(tlv.Stringtobyte(global_cfg.Common.CAMEL.Camel_SCP_id)[0]),
		Camel_SMSAddress: global_cfg.Common.CAMEL.SMSCAddress,
		XVLR:             global_cfg.Common.CAMEL.XVLR,
		ContryCode:       global_cfg.Common.CAMEL.ContryCode,
		OperatorCode:     global_cfg.Common.CAMEL.OperatorCode,
		ResponseFunc:     CamelResponse(),
		RequestFunc:      CamelSend(),
		CamelChannel:     CamelChannel,
	}

	list_listener = tlv.NewListListener()
	camelserver = tlv.NewServer(camel_cfg, list_listener)

	tlv.SetDebug(debugm)

	go camelserver.ServerStart(ctx)
	// Запуск для эксперимента
	//go CamelWriteGorutine(WriteChan)
	//

	// Ждем открытие хотя бы одного соединения
	// Потоки дочерних поднимаются листенером
	for {
		time.Sleep(time.Duration(5) * time.Second)
		if len(list_listener.List) > 0 {
			break
		}
		ProcessInfo("Wait connet to SCP Server")
	}
}

// Горутина записи в поток Camel
// Эксперимент
func CamelWriteGorutine(in chan tlv.WriteStruck) {
	for tmp := range in {
		if _, err = tmp.C.WriteTo(tmp.B); err != nil {
			ProcessError(err)
			if err == io.EOF {
				tmp.C.Close()
				tlv.DeleteCloseConn(tmp.C.Server, camelserver)
				ProcessInfo(tmp.C.RemoteAddr().String() + ": connection close")
				ProcessInfo("Close threads")
			}
			if errors.Is(err, net.ErrClosed) {
				tmp.C.Close()
				tlv.DeleteCloseConn(tmp.C.Server, camelserver)
				ProcessInfo(tmp.C.RemoteAddr().String() + ": connection close")
				ProcessInfo("Close threads")
			}
		}
	}
}

// Отправка сообщений
func CamelWrite(C *tlv.Listener, B []byte) {
	if _, err = C.WriteTo(B); err != nil {
		ProcessError(err)
		if err == io.EOF {
			C.Close()
			tlv.DeleteCloseConn(C.Server, camelserver)
			ProcessInfo(C.RemoteAddr().String() + ": connection close")
			ProcessInfo("Close threads")
		}
		if errors.Is(err, net.ErrClosed) {
			C.Close()
			tlv.DeleteCloseConn(C.Server, camelserver)
			ProcessInfo(C.RemoteAddr().String() + ": connection close")
			ProcessInfo("Close threads")
		}
	}
}

// Горутина записи сообщения по Camel
// Автоматически прописывается brt_id в зависимости от запущенных горутин CamelSend
func CamelSend() tlv.HandReq {
	return func(c *tlv.Listener, in chan tlv.Camel_tcp) {
		for tmprw := range in {
			// Прописываем id BRT
			tmprw.Frame[0x002C].Param[13] = c.BRTId
			tmp, _ := tmprw.Encoder()
			CamelWrite(c, tmp)
			CDRCamelCount.Inc(c.RemoteAddr().String())
		}
	}
}

// Обработчик-ответа Camel
func CamelResponse() tlv.HandOK {
	return func(c *tlv.Listener, camel tlv.Camel_tcp) {
		var err error
		var tmprw []byte
		CDRCamelRCount.Inc(c.RemoteAddr().String())
		switch {
		case camel.Type == tlv.TYPE_AUTHORIZESMS_REJECT:
			sid := string(camel.Frame[0x002C].Param[0:12])
			val := CamelOfflineCDR.Load(sid)
			rr, err := data.CreateCDRRecord(val.RecPool, val.CDRtime, val.Ratio, CDRPatternTask[val.TaskName], val.DstMsisdn, data.RecTypeLACPool{LAC: val.Lac, CELL: val.Cell})
			if err != nil {
				ProcessError(err)
			} else {
				CDRChanneltoFileUni[val.TaskName] <- rr
			}
			CamelOfflineCDR.Delete(sid)
			CDRCamelResponseCount.Inc(fmt.Sprint(int(camel.Frame[0x002C].Param[13])) + " REJECT SMS")
		case camel.Type == tlv.TYPE_AUTHORIZESMS_CONFIRM:
			if camel.Frame[0x0040].Param[0] == byte(0x00) {
				sid := string(camel.Frame[0x002C].Param[0:12])
				val := CamelOfflineCDR.Load(sid)
				rr, err := data.CreateCDRRecord(val.RecPool, val.CDRtime, val.Ratio, CDRPatternTask[val.TaskName], val.DstMsisdn, data.RecTypeLACPool{LAC: val.Lac, CELL: val.Cell})
				if err != nil {
					ProcessError(err)
				} else {
					CDRChanneltoFileUni[val.TaskName] <- rr
				}
				CamelOfflineCDR.Delete(sid)
				CDRCamelResponseCount.Inc(fmt.Sprint(int(camel.Frame[0x002C].Param[13])) + " SMS Charge 00")
			} else {
				camel_req := tlv.NewCamelTCP()
				err = camel_req.EndSMS_req(camel.Frame[0x002C].Param, camelserver)
				if err != nil {
					ProcessError(err)
				}
				tmprw, err = camel_req.Encoder()
				if err != nil {
					ProcessError(err)
				}
				CamelWrite(c, tmprw)
				CDRCamelResponseCount.Inc(fmt.Sprint(int(camel.Frame[0x002C].Param[13])) + " CONFIRM SMS")
			}
		case camel.Type == tlv.TYPE_ENDSMS_RESP:
			// Удаление сессии
			sid := string(camel.Frame[0x002C].Param[0:12])
			CamelOfflineCDR.Delete(sid)
			CDRCamelResponseCount.Inc(fmt.Sprint(int(camel.Frame[0x002C].Param[13])))
		case camel.Type == tlv.TYPE_AUTHORIZEVOICE_CONFIRM:
			if camel.Frame[0x0040].Param[0] == byte(0x00) {
				sid := string(camel.Frame[0x002C].Param[0:12])
				val := CamelOfflineCDR.Load(sid)
				rr, err := data.CreateCDRRecord(val.RecPool, val.CDRtime, val.Ratio, CDRPatternTask[val.TaskName], val.DstMsisdn, data.RecTypeLACPool{LAC: val.Lac, CELL: val.Cell})
				if err != nil {
					ProcessError(err)
				} else {
					CDRChanneltoFileUni[val.TaskName] <- rr
				}
				CamelOfflineCDR.Delete(sid)
				CDRCamelResponseCount.Inc(fmt.Sprint(int(camel.Frame[0x002C].Param[13])) + " SMS Charge 00")
			} else {
				camel_req := tlv.NewCamelTCP()
				err = camel_req.EndVoice_req(camel.Frame[0x002C].Param, camelserver)
				if err != nil {
					ProcessError(err)
				}
				tmprw, err = camel_req.Encoder()
				if err != nil {
					ProcessError(err)
				}
				CamelWrite(c, tmprw)
				CDRCamelResponseCount.Inc(fmt.Sprint(int(camel.Frame[0x002C].Param[13])) + " CONFIRM VOICE")
			}
		case camel.Type == tlv.TYPE_ENDVOICE_RESP:
			// Удаление сессии
			sid := string(camel.Frame[0x002C].Param[0:12])
			CamelOfflineCDR.Delete(sid)
			CDRCamelResponseCount.Inc(fmt.Sprint(int(camel.Frame[0x002C].Param[13])))
		case camel.Type == tlv.TYPE_AUTHORIZEVOICE_REJECT:
			sid := string(camel.Frame[0x002C].Param[0:12])
			val := CamelOfflineCDR.Load(sid)
			rr, err := data.CreateCDRRecord(val.RecPool, val.CDRtime, val.Ratio, CDRPatternTask[val.TaskName], val.DstMsisdn, data.RecTypeLACPool{LAC: val.Lac, CELL: val.Cell})
			if err != nil {
				ProcessError(err)
			} else {
				CDRChanneltoFileUni[val.TaskName] <- rr
			}
			CamelOfflineCDR.Delete(sid)
			CDRCamelResponseCount.Inc(fmt.Sprint(int(camel.Frame[0x002C].Param[13])) + " REJECT VOICE")
		default:
			ProcessInfo(fmt.Sprint("Unknow command ", camel.Type))
			ProcessError(fmt.Sprint("Unknow command ", camel))
		}
	}
}

// Вынесено отдельно для удобства
// Функция отправки диаметр сообщения
func CreateDiamMessage(rec data.RecTypePool, NameTask string, RecType data.RecTypeRatioType) {
	diam_message, sid, err := diameter.CreateCCREventMessage(rec, time.Now(), RecType, dict.Default)
	// Все сообщения добавляются в массив
	// после получения кода 4011 формируется оффлайн CDR
	// Надо понять что передается в интернет сессии к качестве абонента B
	// Возможно будем менять в CreateCCREventMessage
	dst := data.RandomMSISDN(NameTask)
	lc := LACCELLpool[NameTask][rand.Intn(LACCELLlen[NameTask])]
	if err != nil {
		//замьючено так как не везде есть контекст Not use empty ServiceContextId
		//ProcessError(err)
		//Если не смогли сформировать диаметр запрос, шлем оффлайн
		rr, err_cdr := data.CreateCDRRecord(rec, time.Now(), RecType, CDRPatternTask[NameTask], dst, lc)
		if err_cdr != nil {
			ProcessError(err)
		} else {
			CDRChanneltoFileUni[NameTask] <- rr
			CDRPerSec.Inc(NameTask)
		}
	} else {
		// Все сообщения добавляются в массив
		// после получения кода 4011 формируется оффлайн CDR
		BrtOfflineCDR.Store(sid, data.TypeBrtOfflineCdr{RecPool: rec,
			CDRtime:   time.Now(),
			Ratio:     RecType,
			TaskName:  NameTask,
			DstMsisdn: dst,
			Lac:       lc.LAC,
			Cell:      lc.CELL})
		BrtDiamChannel <- diameter.DiamCH{TaskName: NameTask, Message: diam_message}
		//Для БРТ считаем здесь. Пока здесь, похоже фаил пишется дольше
		CDRPerSec.Inc(NameTask)
		CDRPerSecDiam.Inc(NameTask)
	}
}

// Вынесено отдельно для удобства
// Функция отправки кемел сообщения
func CreateCamelMessage(rec data.RecTypePool, NameTask string, RecType data.RecTypeRatioType) {
	camel := tlv.NewCamelTCP()
	dst := data.RandomMSISDN(NameTask)
	lc := LACCELLpool[NameTask][rand.Intn(LACCELLlen[NameTask])]
	if RecType.Record_type == "09" || RecType.Record_type == "08" {
		sid, err := camel.AuthorizeSMS_req(rec.Msisdn, rec.IMSI, RecType.Record_type, dst, lc, camelserver)

		if err == nil {
			CamelOfflineCDR.Store(string(sid), data.TypeBrtOfflineCdr{RecPool: rec,
				CDRtime:   time.Now(),
				Ratio:     RecType,
				TaskName:  NameTask,
				DstMsisdn: dst,
				Lac:       lc.LAC,
				Cell:      lc.CELL})
			CamelChannel <- camel
			CDRPerSec.Inc(NameTask)
			CDRPerSecCamel.Inc(NameTask)
		} else {
			ProcessError(err)
		}
	} else if RecType.Record_type == "01" || RecType.Record_type == "02" {
		sid, err := camel.AuthorizeVoice_req(rec.Msisdn, rec.IMSI, RecType.Record_type, dst, lc, camelserver)

		if err == nil {
			CamelOfflineCDR.Store(string(sid), data.TypeBrtOfflineCdr{RecPool: rec,
				CDRtime:   time.Now(),
				Ratio:     RecType,
				TaskName:  NameTask,
				DstMsisdn: dst,
				Lac:       lc.LAC,
				Cell:      lc.CELL})
			CamelChannel <- camel
			CDRPerSec.Inc(NameTask)
			CDRPerSecCamel.Inc(NameTask)
		} else {
			ProcessError(err)
		}
	} else {
		// Если не прописаны типы сформировать оффлайн
		rr, err := data.CreateCDRRecord(rec, time.Now(), RecType, CDRPatternTask[NameTask], dst, lc)
		if err != nil {
			ProcessError(err)
		} else {
			CDRChanneltoFileUni[NameTask] <- rr
			CDRPerSec.Inc(NameTask)
		}
	}
}

// Запуск Диаметра и Кемел
func StartClient() {
	// Запуск БРТ
	if brt && len(brtlist) != 0 {
		// Запуск потока БРТ
		// Горутина запускается из функции, по количеству серверов
		// Поток идет только по правилу один хост один пир
		StartDiameterClient()
	}

	// Запуск Camel
	if camel {
		// Запуск потока Camel
		// Горутина запускается из функции, по количеству серверов и потоков
		StartCamelServer()
	}
}

func init() {

}

func end() {
	ProcessInfo("Start schelduler")
	// Ждем выполнение таймаута
	// Добавить в дальнейшем выход по событию от системы
	go func() {
		<-time.After(time.Duration(global_cfg.Common.Duration) * time.Second)
		stop()
	}()

	// Ждем получения от контекста о завершении
	<-ctx.Done()
	ProcessInfo(ctx.Err().Error())
	ProcessInfo("Stoping")
	gostop = true

	// Закрытие окрытого порта
	if camel {
		camelserver.ServerStop()
	}

	// Задержка остановки, Ждем досылки ответов
	kk := 0
	for {
		time.Sleep(1 * time.Second)
		kk++
		ProcessInfo("Stoping " + strconv.Itoa(kk) + "s ")
		if kk > 10 {
			break
		}
	}

	// Вывод статистики работы утилиты
	for _, thread := range global_cfg.Tasks {
		ProcessInfo("All load " + strconv.Itoa(CDRRecCount.Load(thread.Name)) + " records for " + thread.Name)
		if tofile {
			ProcessInfo("Save " + strconv.Itoa(CDRFileCount.Load(thread.Name)) + " files for " + thread.Name)
		}
		if brt || camel {
			ProcessInfo("Save offline " + strconv.Itoa(CDRRecCount.Load(thread.Name+"offline")) + " CDR for " + thread.Name)
		}
	}
	if brt {
		for _, ip := range global_cfg.Common.BRT {
			ProcessInfo("Send " + strconv.Itoa(CDRDiamCount.Load(ip)) + " Diameter messages to " + ip)
		}
		CDRDiamResponseCount.LoadRangeToLogFunc("Diameter response code ", ProcessInfo)
	}
	if camel {
		CDRCamelCount.LoadRangeToLogFunc("Camel messages ", ProcessInfo)
		CDRCamelResponseCount.LoadRangeToLogFunc("Camel response code BRT id ", ProcessInfo)
	}

	ProcessInfo("End schelduler")

	//Удалить
	if brt || camel {

		if len(BrtOfflineCDR.CDROffline) > 0 {
			ProcessDebug(len(BrtOfflineCDR.CDROffline))
			ProcessDebug(BrtOfflineCDR.Random())
			CDRDiamRCount.LoadRangeToLogFunc("BRT messages revc ", ProcessInfo)
		}
		if len(CamelOfflineCDR.CDROffline) > 0 {
			ProcessDebug(len(CamelOfflineCDR.CDROffline))
		}
		CDRCamelRCount.LoadRangeToLogFunc("Camel messages revc ", ProcessInfo)
	}

	// Очищаем директории
	if rm {
		ProcessInfo("Remove CDR files")
		delfilefortest()
	}

	err = pid.RemovePID(pidFileName)
	if err != nil {
		ProcessError("Can not remove pid-file: " + err.Error())
	}
	ProcessInfo("Remove PID file")
}
