package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

//Power by  Egor Kovalchuk

// логи
const logFileName = "generator.log"
const pidFileName = "generator.pid"

//конфиг
//var cfg pdata.Config

// режим работы сервиса(дебаг мод)
var debugm bool
var emul bool

// ошибки
var err error

// режим работы сервиса
var startdaemon bool

// запрос версии
var version bool

/*
Vesion 0.1
Create
*/
const versionutil = "0.1"

func main() {

	//start program
	var argument string
	/*var progName string

	progName = os.Args[0]*/

	if os.Args != nil && len(os.Args) > 1 {
		argument = os.Args[1]
	} else {
		helpstart()
		return
	}

	if argument == "-h" {
		helpstart()
		return
	}

	flag.BoolVar(&debugm, "t", false, "a bool")
	flag.BoolVar(&startdaemon, "d", false, "a bool")
	flag.BoolVar(&version, "v", false, "a bool")
	flag.BoolVar(&emul, "e", false, "a bool")
	// for Linux compile
	stdaemon := flag.Bool("s", false, "a bool") // для передачи
	// --for Linux compile
	var listname string
	flag.StringVar(&listname, "l", "", "Name list is not empty")
	var message string
	flag.StringVar(&message, "m", "", "Messge is not empty")
	flag.Parse()

	if startdaemon {
		filer, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatal(err)
		}

		log.SetOutput(filer)
		log.Println("- - - - - - - - - - - - - - -")
		log.Println("Start daemon mode")
		if debugm {
			log.Println("Start with debug mode")
		}

		fmt.Println("Start daemon mode")
	}

	//load conf
	//readconf(&cfg, "smsc.ini")

	if version {
		fmt.Println("Version utils " + versionutil)
		return
	}

	if startdaemon || *stdaemon {

		//processinghttp(&cfg, debugm)

		log.Println("daemon terminated")

	} else {

		StartShellMode(message, listname)

	}
	fmt.Println("Done")
	return

}

func processError(err error) {
	fmt.Println(err)
	os.Exit(2)
}

func readconf( /*cfg *pdata.Config,*/ confname string) {
	/*	file, err := os.Open(confname)
		if err != nil {
			processError(err)
		}

		decoder := json.NewDecoder(file)
		err = decoder.Decode(&cfg)
		if err != nil {
			processError(err)
		}

		file.Close()

		if cfg.IPRestrictionType != 0 {
			var nets []net.IPNet

			for _, p := range cfg.IPRestriction {

				n, err := iprest.IPRest(p)
				if err != nil {
					logwrite(err)
				} else {
					nets = append(nets, n)
				}
			}
			cfg.Nets = nets
		}*/
}

// StartShellMode запуск в режиме скрипта
func StartShellMode(message string, listname string) {}

func helpstart() {
	fmt.Println("Use -l Name list -m \"Text message\"")
	fmt.Println("Use -d start deamon mode(HTTP service)")
	fmt.Println("Example 1 curl localhost:8080 -X GET -F src=IT -F lst=rss_1 -F text=hello")
	fmt.Println("Example 2 curl localhost:8080 -X GET -F src=IT -F dst=79XXXXXXXX -F text=hello)")
	fmt.Println("Example 3 curl localhost:8080/conf -X GET -F reloadconf=1")
	fmt.Println("Example 4 curl localhost:8080/list -X GET ")
	fmt.Println("Use -s stop deamon mode(HTTP service)")
	fmt.Println("Use -t start with debug mode")
}

func logwrite(err error) {
	if startdaemon {
		log.Println(err)
	} else {
		fmt.Println(err)
	}
}
