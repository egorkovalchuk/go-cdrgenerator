package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
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

	m sync.Mutex

	CDRPerSec         = data.NewCounters()
	CDRChanneltoFile1 = make(map[string](chan string))

	CDRChanneltoFile     = make(chan string)
	CDRRoamChanneltoFile = make(chan string)

	Flag string

/*
Vesion 0.1
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

	if cfg.Common.Duration == 0 {
		log.Println("Script use default duration - 14400 sec")
		cfg.Common.Duration = 14400
	}

	// запуск горутины записи в лог
	go LogWriteForGoRutine(ErrorChannel)
	go Monitor()
	// Обнуляем счетчик

	for _, task := range cfg.Tasks {
		//	CDRPerSec[task.Name] = 0
		CDRPerSec.Store(task.Name, 0)
		CDRChanneltoFile1[task.Name] = make(chan string)
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
	// родительские породождают дочерние по формуле ???
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
					//Есть идея использовать массив из канало?
					//Если пишем в фаил
					if tofile {
						if thread.Name == "local" {
							go StartFileCDR(thread.PathsToSave[0]+thread.Template_save_file, CDRChanneltoFile)
						} else {
							go StartFileCDR(thread.PathsToSave[0]+thread.Template_save_file, CDRRoamChanneltoFile)
						}
						//go StartFileCDR(thread.PathsToSave[0]+thread.Template_save_file, CDRChanneltoFile1[thread.Name])
					}

					go StartTask(PoolList, thread)
				}

			}
		}
	}

	log.Println("Start schelduler")
	time.Sleep(time.Duration(cfg.Common.Duration) * time.Second)
	log.Println("End schelduler")

}

// Функция контроля рейтов
func Monitor() {
	heartbeat := time.Tick(1 * time.Second)
	var CDR int
	time.Sleep(5 * time.Second)

	for {
		select {
		case <-heartbeat:
			for _, thread := range cfg.Tasks {
				CDR = CDRPerSec.Load(thread.Name)
				log.Println("Speed task " + thread.Name + " " + strconv.Itoa(CDR) + " op/s")
				if CDR < thread.CallsPerSecond {
					m.Lock()
					Flag = thread.Name
					m.Unlock()
				}

				CDRPerSec.Store(thread.Name, 0)

			}
		}
	}
}

// Горутина формирования CDR
func StartTask(PoolList []data.RecTypePool, cfg data.TasksType) {

	var PoolIndex int
	var PoolIndexMax int
	var CDR int
	PoolIndex = 0
	PoolIndexMax = len(PoolList) - 1

	for {
		/*m.Lock()
		if Flag == cfg.Name {
			log.Println("Start new thead " + cfg.Name)
			go StartTask(PoolList, cfg)
			//	m.Lock()
			Flag = ""
			//	m.Unlock()
		}
		m.Unlock()*/

		CDR = CDRPerSec.Load(cfg.Name)
		if CDR < cfg.CallsPerSecond {
			// Сброс счетчика
			if PoolIndex >= PoolIndexMax {
				PoolIndex = 0
			}

			CDRPerSec.Inc(cfg.Name)
			PoolIndex++

			rr := data.CreateCDRRecord(PoolList[PoolIndex], time.Now(), cfg.RecTypeRatio[0], cfg.CDR_pattern)

			if cfg.Name == "local" {
				CDRChanneltoFile <- rr
			} else {
				CDRRoamChanneltoFile <- rr
			}
			if err != nil {
				ErrorChannel <- err
			}
		}

	}

}

// Запись в Фаил
func StartFileCDR(FileName string, InputString <-chan string) {
	f, err := os.Create(strings.Replace(FileName, "{date}", time.Now().Format("20060201030405"), 1))

	if err != nil {
		ErrorChannel <- err
	}
	log.Println("Start write " + f.Name())
	defer f.Close()

	//heartbeat := time.Tick(1 * time.Second)
	//Переписать на создание нового файла каждую Х секнду
	heartbeat := time.Tick(2 * time.Second)

	for {
		//for str := range InputString {
		select {
		case <-heartbeat:
			f.Close()
			f, err = os.Create(strings.Replace(FileName, "{date}", time.Now().Format("20060201030405"), 1))
			if err != nil {
				ErrorChannel <- err
			}
			defer f.Close() //Закрыть фаил при нешаттном завершении
		default:
			//MapMutex.RLock()
			str := <-InputString
			//MapMutex.RUnlock()
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
