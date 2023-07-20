package main

import (
	"fmt"

	"log"
	"os"
	"time"

	"github.com/egorkovalchuk/go-cdrgenerator/data"
	"github.com/fiorix/go-diameter/v4/diam"
)

//Просто запись в лог
func LogWrite(err error) {
	if startdaemon {
		log.Println(err)
	} else {
		fmt.Println(err)
	}
}

// Запись ошибок из горутин
func LogWriteForGoRutine(err chan error) {
	for err := range err {
		datetime := time.Now().Local().Format("2006/02/01 15:04:05 ")
		log.SetPrefix(datetime + "ERROR: ")
		log.SetFlags(0)
		log.Println(err)
		log.SetPrefix("")
		log.SetFlags(log.Ldate | log.Ltime)
	}
}

// Запись ошибок из горутин для диаметра
func DiamPrintErrors(ec <-chan *diam.ErrorReport) {
	for err := range ec {
		datetime := time.Now().Local().Format("2006/02/01 15:04:05 ")
		log.SetPrefix(datetime + "DIAM: ")
		log.SetFlags(0)
		log.Println(err)
		log.SetPrefix("")
		log.SetFlags(log.Ldate | log.Ltime)
	}
}

// Запись в лог при включенном дебаге
// Сделать горутиной?
func ProcessDebug(logtext interface{}) {
	if debugm {
		// изменить интерыейс?
		datetime := time.Now().Local().Format("2006/02/01 15:04:05 ")
		log.SetPrefix(datetime + "DEBUG: ")
		log.SetFlags(0)
		log.Println(logtext)
		log.SetPrefix("")
		log.SetFlags(log.Ldate | log.Ltime)
	}
}

// Нештатное завершение при критичной ошибке
func ProcessError(err error) {
	fmt.Println(err)
	os.Exit(2)
}

// Инициализация переменных
func InitVariables() {
	//Если не задан параметр используем дефольтное значение
	if global_cfg.Common.Duration == 0 {
		log.Println("Script use default duration - 14400 sec")
		global_cfg.Common.Duration = 14400
	}

	// Обнуляем счетчик и инициализируем
	for _, task := range global_cfg.Tasks {
		// Иницмализация счетчика
		CDRPerSec.Store(task.Name, 0)
		// Инициализация флага запуска дополнительной горутины
		Flag.Store(task.Name, 0)
		//Инициализация каналов
		CDRChanneltoFileUni[task.Name] = make(chan string)
		CDRChanneltoBRTUni[task.Name] = make(chan string)

		//Добавлено для тестов, по идее использовать CDRChanneltoBRTUni
		BrtDiamChannelAnswer = make(chan struct{}, 1000)
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
		CDRBRTCount.Store(ip, 0)
	}

}
