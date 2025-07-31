package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

type LogStruct struct {
	t    string
	text interface{}
}

type LogWriter struct {
	logger      *log.Logger
	filer       *os.File
	LogChannel  chan LogStruct
	logFileName string
	debugm      bool
}

func NewLogWriter(logFileName string, debugm bool) *LogWriter {
	return &LogWriter{
		logFileName: logFileName,
		debugm:      debugm,
		LogChannel:  make(chan LogStruct),
	}
}

// Запись ошибок из горутин
// можно добавить ротейт по дате + архив в отдельном потоке
func (l *LogWriter) LogWriteForGoRutineStruct() {
	filer, err := os.OpenFile(l.logFileName, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer filer.Close()
	l.logger = log.New(filer, "", 0)

	for entry := range l.LogChannel {
		prefix := time.Now().Local().Format("2006/01/02 15:04:05") + " " + entry.t + ": "
		l.logger.SetPrefix(prefix)
		l.logger.Println(entry.text)
	}
}

// Запись в лог при включенном дебаге
func (l *LogWriter) ProcessDebug(logtext interface{}) {
	if l.debugm {
		l.LogChannel <- LogStruct{"DEBUG", logtext}
	}
}

// Запись в лог ошибок
func (l *LogWriter) ProcessError(logtext interface{}) {
	l.LogChannel <- LogStruct{"ERROR", logtext}
}

// Запись в лог ошибок cсо множеством переменных
func (l *LogWriter) ProcessErrorAny(logtext ...interface{}) {
	t := ""
	for _, a := range logtext {
		t += fmt.Sprint(a) + " "
	}
	l.LogChannel <- LogStruct{"ERROR", t}
}

// Запись в лог WARM
func (l *LogWriter) ProcessWarm(logtext interface{}) {
	l.LogChannel <- LogStruct{"WARM", logtext}
}

// Запись в лог INFO
func (l *LogWriter) ProcessInfo(logtext interface{}) {
	l.LogChannel <- LogStruct{"INFO", logtext}
}

// Запись в лог Diam
func (l *LogWriter) ProcessDiam(logtext interface{}) {
	l.LogChannel <- LogStruct{"DIAM", logtext}
}

// Запись в лог Influx
func (l *LogWriter) ProcessInflux(logtext interface{}) {
	l.LogChannel <- LogStruct{"INFLUX", logtext}
}

// Нештатное завершение при критичной ошибке
func (l *LogWriter) ProcessPanic(logtext interface{}) {
	fmt.Println(logtext)
	os.Exit(2)
}

// Смена уровня логирования
func (l *LogWriter) ChangeDebugLevel(debugm bool) {
	l.debugm = debugm
}

func (l *LogWriter) GetLogger() *log.Logger {
	return l.logger
}

// Запись в лог
func (l *LogWriter) ProcessLog(level string, logtext interface{}) {
	switch level {
	case "DEBUG":
		l.ProcessDebug(logtext)
	case "PANIC":
		l.ProcessPanic(logtext)
	default:
		l.LogChannel <- LogStruct{level, logtext}
	}
}
