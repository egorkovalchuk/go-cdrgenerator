package main

import (
	"encoding/csv"
	"fmt"
	"strconv"

	"log"
	"os"
	"time"

	"github.com/egorkovalchuk/go-cdrgenerator/data"
	"github.com/fiorix/go-diameter/v4/diam"
)

type LogStruct struct {
	t    string
	text interface{}
}

// Просто запись в лог
func LogWrite(err error) {
	if startdaemon {
		ProcessError(err)
	} else {
		fmt.Println(err)
	}
}

// Запись ошибок из горутин
// можно добавить ротейт по дате + архив в отдельном потоке
func LogWriteForGoRutineStruct(err chan LogStruct) {
	for i := range err {
		datetime := time.Now().Local().Format("2006/01/02 15:04:05")
		log.SetPrefix(datetime + " " + i.t + ": ")
		log.SetFlags(0)
		log.Println(i.text)
		log.SetPrefix("")
		log.SetFlags(log.Ldate | log.Ltime)
	}
}

// Запись ошибок из горутин для диаметра
func DiamPrintErrors(ec <-chan *diam.ErrorReport) {
	for err := range ec {
		LogChannel <- LogStruct{"DIAM", err}
	}
}

// Запись в лог при включенном дебаге
// Сделать горутиной?
func ProcessDebug(logtext interface{}) {
	if debugm {
		LogChannel <- LogStruct{"DEBUG", logtext}
	}
}

// Запись в лог ошибок
func ProcessError(logtext interface{}) {
	LogChannel <- LogStruct{"ERROR", logtext}
}

// Запись в лог ошибок cсо множеством переменных
func ProcessErrorAny(logtext ...interface{}) {
	t := ""
	for _, a := range logtext {
		t += fmt.Sprint(a) + " "
	}
	LogChannel <- LogStruct{"ERROR", t}
}

// Запись в лог WARM
func ProcessWarm(logtext interface{}) {
	LogChannel <- LogStruct{"WARM", logtext}
}

// Запись в лог INFO
func ProcessInfo(logtext interface{}) {
	LogChannel <- LogStruct{"INFO", logtext}
}

// Запись в лог Diam
func ProcessDiam(logtext interface{}) {
	LogChannel <- LogStruct{"DIAM", logtext}
}

// Запись в лог Camel
func ProcessCamel(logtext interface{}) {
	LogChannel <- LogStruct{"CAMEL", logtext}
}

// Запись в лог Influx
func ProcessInflux(logtext interface{}) {
	LogChannel <- LogStruct{"INFLUX", logtext}
}

// Нештатное завершение при критичной ошибке
func ProcessPanic(logtext interface{}) {
	fmt.Println(logtext)
	os.Exit(2)
}

// Инициализация переменных
func InitVariables() {
	//Если не задан параметр используем дефолтное значение
	if global_cfg.Common.Duration == 0 {
		ProcessInfo("Script use default duration - 14400 sec")
		global_cfg.Common.Duration = 14400
	}

	time_sleep = 1000000 / global_cfg.Common.Duration

	// Обнуляем счетчик и инициализируем
	for _, task := range global_cfg.Tasks {
		// Иницмализация счетчика
		CDRPerSec.Store(task.Name, 0)
		CDRPerSecCamel.Store(task.Name, 0)
		CDRPerSecDiam.Store(task.Name, 0)
		// Инициализация флага запуска дополнительной горутины
		Flag.Store(task.Name, 0)
		// Инициализация каналов
		CDRChanneltoFileUni[task.Name] = make(chan string)
		// Инициализация CDR_pattern
		CDRPatternTask[task.Name] = data.CDRPatternType{
			Pattern: task.CDR_pattern,
			MsisdnB: task.DefaultMSISDN_B}

		//Добавлено для тестов, по идее использовать CDRChanneltoBRTUni
		BrtDiamChannelAnswer = make(chan diam.Message, 1000)
		BrtDiamChannel = make(chan data.DiamCH, 1000)

		// Заполнение интервалов для радндомайзера
		// Инициализация среза для полсчета типов
		task.RecTypeRatio[0].RangeMax = task.RecTypeRatio[0].Rate
		task.RecTypeRatio[0].RangeMin = 0
		for i := 1; i < len(task.RecTypeRatio); i++ {
			// Заполняем проценты попадания типа звонков
			// 0..56..78..98..100
			// в основном теле генерируем случайное значение от 0 до 100 которое должно попасть с один из интервалов
			task.RecTypeRatio[i].RangeMin = task.RecTypeRatio[i-1].RangeMax
			task.RecTypeRatio[i].RangeMax = task.RecTypeRatio[i].Rate + task.RecTypeRatio[i].RangeMin
			// Нахрена это добавлено? пока не удаляю. мож вспомню
			// Flag.Store(task.Name+" "+task.RecTypeRatio[i].Name, 0)
			// Инициализация счетчика типов звонка
			CDRRecTypeCount.AddMap(task.Name, task.RecTypeRatio[i].Name, 0)
		}
	}

	// Счетчик записи в БРТ
	for _, ip := range global_cfg.Common.BRT {
		CDRDiamCount.Store(ip, 0)
	}

	// Зачитывание LAC/CELL
	for _, task := range global_cfg.Tasks {
		if task.DatapoolCsvLac != "" {
			f, err := os.Open(task.DatapoolCsvLac)
			if err != nil {
				ProcessErrorAny("Unable to read input file "+task.DatapoolCsvLac, err)
				ProcessError("Thread " + task.Name + " not start")
			} else {
				defer f.Close()
				ProcessDebug("Start load" + task.DatapoolCsvLac)

				// read csv values using csv.Reader
				csvReader := csv.NewReader(f)
				// Разделитель CSV
				csvReader.Comma = ';'
				csv, err := csvReader.ReadAll()
				if err != nil {
					ProcessError(err)
				} else {
					var PoolList []data.RecTypeLACPool
					for i, line := range csv {
						if i > 0 { // omit header line
							var rec data.RecTypeLACPool
							rec.LAC, _ = strconv.Atoi(line[0])
							rec.CELL, _ = strconv.Atoi(line[1])
							PoolList = append(PoolList, rec)
						}
					}
					LACCELLpool[task.Name] = PoolList
					LACCELLlen[task.Name] = len(PoolList)
				}
			}
		} else {
			ProcessInfo("Pool not defined for " + task.Name)
		}
		if len(LACCELLpool[task.Name]) == 0 {
			LACCELLpool[task.Name] = append(LACCELLpool[task.Name], data.RecTypeLACPool{LAC: task.DefaultLAC, CELL: task.DefaultCELL})
			LACCELLlen[task.Name] = len(LACCELLpool[task.Name])
		}
	}
}

// Аналог Sleep.
func sleep(d time.Duration) {
	<-time.After(d)
}

func delfilefortest() {
	// Удаляем файлы в директорий их конфига
	for _, i := range global_cfg.Tasks {
		for _, j := range i.PathsToSave {
			directory := j
			readDirectory, _ := os.Open(directory)
			allFiles, _ := readDirectory.Readdir(0)

			for f := range allFiles {
				file := allFiles[f]

				fileName := file.Name()
				filePath := directory + fileName

				os.Remove(filePath)
			}
		}
	}
}
