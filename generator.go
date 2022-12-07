package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/egorkovalchuk/go-cdrgenerator/data"
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

	flag.BoolVar(&debugm, "t", false, "a bool")
	flag.BoolVar(&startdaemon, "d", false, "a bool")
	flag.BoolVar(&version, "v", false, "a bool")
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

	if startdaemon || *stdaemon {

		//processinghttp(&cfg, debugm)

		log.Println("daemon terminated")

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
	for _, thread := range cfg.Tasks {
		if thread.DatapoolCsvFile == "" {
			log.Println("Тo file name specified for" + thread.Name)
		} else {
			f, err := os.Open(thread.DatapoolCsvFile)
			if err != nil {
				log.Println("Unable to read input file "+thread.DatapoolCsvFile, err)
				log.Println("Thread " + thread.Name + " not start")
			} else {
				defer f.Close()
				//Вынести в глобальные?

				// read csv values using csv.Reader
				csvReader := csv.NewReader(f)
				csv, err := csvReader.ReadAll()
				PoolList := data.CreatePoolList(csv)
				if err != nil {
					log.Println(err)
				} else {
					if debugm {
						log.Println(PoolList[len(PoolList)-1])
					}
					log.Println("Load " + strconv.Itoa(len(PoolList)) + " records")
					log.Println("Start thread for " + thread.Name)
					go StartTask(PoolList, thread)
				}

			}
		}
	}

	log.Println("Start schelduler")
	time.Sleep(time.Duration(cfg.Common.Duration) * time.Second)
	log.Println("End schelduler")

}

// Горутина формирования CDR
func StartTask(PoolList []data.RecTypePool, cfg data.TasksType) {
	for i := 0; i < 5; i++ {
		log.Println(i)
		time.Sleep(3 * time.Second)
	}

}

// Чтение из потока файлов (работа без генератора)
func StartTransferCDR() {

}

// Запуск потоков подключения к БРТ
func StartTelnet() {
	/*diam_cfg := &sm.Settings
	println(diam_cfg)*/
}

// Поток телнета
// два типа каналов CDR и закрытие/переоткрытие
// +надо сделать keepalive
func StartTheadTelnet() {

}

func logwrite(err error) {
	if startdaemon {
		log.Println(err)
	} else {
		fmt.Println(err)
	}
}
