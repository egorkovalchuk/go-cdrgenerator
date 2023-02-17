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
	"github.com/fiorix/go-diameter/diam/datatype"
	"github.com/fiorix/go-diameter/diam/sm"
)

//Power by  Egor Kovalchuk

const (
	logFileName = "generator.log"
	pidFileName = "generator.pid"
	versionutil = "0.1"
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

	// для теста коннекта
	brttest bool

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
	//Статистика записи
	CDRRecCount     = data.NewCounters()
	CDRFileCount    = data.NewCounters()
	CDRRecTypeCount = data.NewRecTypeCounters()

	//Не используется(в коде закоменчено)
	CDRChanneltoFile     = make(chan string)
	CDRRoamChanneltoFile = make(chan string)

	m sync.Mutex

/*
Vesion 0.2
Create
*/

)

// Запись в лог при включенном дебаге
func ProcessDebug(logtext interface{}) {
	if debugm {
		log.Println(logtext)
	}
}

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
	//Временная переменная для проверки
	flag.BoolVar(&brttest, "brt", false, "Test connect to BRT")
	//Временная переменная для проверки
	// for Linux compile
	stdaemon := flag.Bool("s", false, "a bool") // для передачи
	// --for Linux compile
	flag.Parse()

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

	// запуск горутины записи в лог
	go LogWriteForGoRutine(ErrorChannel)
	// Запуск мониторинга
	go Monitor()

	// Обнуляем счетчик и инициализируем
	for _, task := range cfg.Tasks {
		CDRPerSec.Store(task.Name, 0)
		Flag.Store(task.Name, 0)
		CDRChanneltoFileUni[task.Name] = make(chan string)
		// Заполнение интервалов для радндомайзера
		task.RecTypeRatio[0].RangeMax = task.RecTypeRatio[0].Rate
		task.RecTypeRatio[0].RangeMin = 0
		for i := 1; i < len(task.RecTypeRatio); i++ {
			task.RecTypeRatio[i].RangeMin = task.RecTypeRatio[i-1].RangeMax
			task.RecTypeRatio[i].RangeMax = task.RecTypeRatio[i].Rate + task.RecTypeRatio[i].RangeMin
			Flag.Store(task.Name+" "+task.RecTypeRatio[i].Name, 0)
		}

	}

	if startdaemon || *stdaemon {

		//processinghttp(&cfg, debugm)

		log.Println("daemon terminated")

	} else if brttest {
		StartDiameterClient()
	} else {

		StartSimpleMode()

	}
	fmt.Println("Done")
	return

}

func processError(err error) {
	fmt.Println(err)
	os.Exit(2)
}

func readconf(cfg *data.Config, confname string) {
	file, err := os.Open(confname)
	if err != nil {
		processError(err)
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		processError(err)
	}

	file.Close()

}

// StartSimpleMode запуск в режиме скрипта
func StartSimpleMode() {
	// запускаем отдельные потоки родительские потоки для задач из конфига
	// родительские породождают дочерние по формуле
	// при завершении времени останавливают дочерние и сами

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
				if debugm {
					log.Print("Start load" + thread.DatapoolCsvFile)
				}
				// read csv values using csv.Reader
				csvReader := csv.NewReader(f)
				csvReader.Comma = ';'
				csv, err := csvReader.ReadAll()

				var PoolList data.PoolSubs
				PoolList = PoolList.CreatePoolList(csv, thread)

				if err != nil {
					log.Println(err)
				} else {
					if debugm {
						log.Print("Last record ")
						log.Println(PoolList[len(PoolList)-1])

					}
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
				if debugm {
					log.Println("Speed task " + thread.Name + " " + strconv.Itoa(CDR) + " op/s")
				}
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
					log.Println("Load " + thread.Name + " calls type " + t.Name + " " + CDRRecTypeCount.LoadString(thread.Name+" "+t.Name))
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

			//CDRPerSec.Inc(cfg.Name)
			PoolIndex++

			tmp = PoolList[PoolIndex].CallsCount

			if tmp != 0 {

				PoolList[PoolIndex].CallsCount = tmp - 1

				RecTypeIndex = data.RandomRecType(cfg.RecTypeRatio, rand.Intn(100))
				CDRRecTypeCount.Inc(cfg.Name + " " + cfg.RecTypeRatio[RecTypeIndex].Name)

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
func StartFileCDR(task data.TasksType, InputString <-chan string) {
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
			if debugm {
				log.Println("Start write " + f.Name())
			}
			CDRFileCount.Inc(task.Name)
			defer f.Close() //Закрыть фаил при нешаттном завершении
		default:

			str := <-InputString
			//Перенес из генерации, в одном потоке будет работать быстрее
			CDRPerSec.Inc(task.Name)

			_, err = f.WriteString(str)
			_, err = f.WriteString("\n")

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
	brt_adress = append(brt_adress, datatype.Address(net.ParseIP(cfg.Common.BRT)))

	diam_cfg := &sm.Settings{
		OriginHost:       datatype.DiameterIdentity("client"),
		OriginRealm:      datatype.DiameterIdentity("go-diameter"),
		VendorID:         data.PETER_SERVICE_VENDOR_ID,
		ProductName:      "CDR-generator",
		OriginStateID:    datatype.Unsigned32(time.Now().Unix()),
		FirmwareRevision: 1,
		HostIPAddresses:  brt_adress,
	}

	// Create the state machine (it's a diam.ServeMux) and client.
	mux := sm.New(diam_cfg)
	log.Println(mux.Settings())

}

// Поток телнета
// два типа каналов CDR и закрытие/переоткрытие
// +надо сделать keepalive
func StartDiameterTelnet() {

}

//Просто запись в лог
func logwrite(err error) {
	if startdaemon {
		log.Println(err)
	} else {
		fmt.Println(err)
	}
}

// Поток телнета Кемел
// два типа каналов CDR и закрытие/переоткрытие
// +надо сделать keepalive
func StartCamelTelnet() {

}

// Запись ошибок из горутин
func LogWriteForGoRutine(err <-chan error) {
	for err := range err {
		log.Println(err)
	}
}

// Запись ошибок из горутин для диаметра
func printErrors(ec <-chan *diam.ErrorReport) {
	for err := range ec {
		log.Println(err)
	}
}
