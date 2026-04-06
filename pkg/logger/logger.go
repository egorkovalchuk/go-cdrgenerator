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
func (l *LogWriter) ProcessDebug(logtext interface{}, opt ...string) {
	if l.debugm {
		l.LogChannel <- LogStruct{l.Prefix(opt...) + "DEBUG", logtext}
	}
}

// Запись в лог ошибок
func (l *LogWriter) ProcessError(logtext interface{}, opt ...string) {
	l.LogChannel <- LogStruct{l.Prefix(opt...) + "ERROR", logtext}
}

// Запись в лог ошибок
func (l *LogWriter) ProcessCritical(logtext interface{}, opt ...string) {
	l.LogChannel <- LogStruct{l.Prefix(opt...) + "CRITICAL", logtext}
	fmt.Println(logtext)
}

// Запись в лог ошибок cсо множеством переменных
func (l *LogWriter) ProcessErrorAny(logtext ...interface{}) {
	t := ""
	for _, a := range logtext {
		t += fmt.Sprint(a) + " "
	}
	l.LogChannel <- LogStruct{l.Prefix() + "ERROR", t}
}

// Запись в лог WARM
func (l *LogWriter) ProcessWarm(logtext interface{}, opt ...string) {
	l.LogChannel <- LogStruct{l.Prefix(opt...) + "WARM", logtext}
}

// Запись в лог INFO
func (l *LogWriter) ProcessInfo(logtext interface{}, opt ...string) {
	l.LogChannel <- LogStruct{l.Prefix(opt...) + "INFO", logtext}
}

// Нештатное завершение при критичной ошибке
func (l *LogWriter) ProcessPanic(logtext interface{}) {
	fmt.Println(logtext)
	os.Exit(2)
}

// Смена уровня логирования
func (l *LogWriter) ChangeDebugLevel(debugm bool) {
	l.debugm = debugm
	if debugm {
		l.LogChannel <- LogStruct{l.Prefix() + "DEBUG", "Start debug mode"}
	} else {
		l.LogChannel <- LogStruct{l.Prefix() + "DEBUG", "Stop debug mode"}
	}
}

// Уровень логирования
func (l *LogWriter) GetDebugLevel() bool {
	return l.debugm
}

func (l *LogWriter) GetLogger() *log.Logger {
	return l.logger
}

// Запись в лог
func (l *LogWriter) ProcessLog(level string, logtext interface{}, opt ...string) {
	switch level {
	case "DEBUG":
		l.ProcessDebug(logtext, opt...)
	case "PANIC":
		l.ProcessPanic(logtext)
	default:
		l.LogChannel <- LogStruct{l.Prefix(opt...) + level, logtext}
	}
}

func (l *LogWriter) Printf(level, format string, args ...any) {
	l.ProcessLog(level, fmt.Sprintf(format, args...))
}

func (l *LogWriter) Prefix(opt ...string) (prefix string) {
	if len(opt) > 0 {
		for ind, i := range opt {
			switch ind {
			case 0:
				prefix += fmt.Sprintf("[%s] ", i)
			case 1:
				prefix += fmt.Sprintf("(%s) ", i)
			default:
				prefix += i + " "
			}
		}
	} else {
		prefix = "[MAIN] "
	}
	return
}
