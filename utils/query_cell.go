//go:build ignore || gen
// +build ignore gen
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "github.com/sijms/go-ora/v2"
)

type UtilConf struct {
	Tasks []TasksUtilType
}

type TasksUtilType struct {
	Name          string `json:"Name"`
	MacrID        int    `json:"Macr_id"`
	Region        string `json:"Region"`
	FileName      string `json:"FileName"`
	Query         string `json:"Query"`
	ConnectString string `json:"ConnectString"`
}

// Запуск утилиты генерации пула LAC/CELL
var (
	pool             bool
	connetion_string string
	pool_task        string
	mass             bool
	cfg              UtilConf
)

func main() {
	// Утилиты
	flag.BoolVar(&pool, "pool", false, "Starting pool creation LAC/CELL, use -t task name -p password")
	flag.StringVar(&pool_task, "t", "", "Task Name")
	flag.StringVar(&connetion_string, "p", "", "Password")
	flag.BoolVar(&mass, "m", false, "Start all task")
	flag.Parse()

	// Проверка на доп параметры
	if pool_task == "" {
		fmt.Println("Stop utils. Task name is empty. Use -t")
		return
	}
	if connetion_string == "" {
		fmt.Println("Stop utils. Password is empty. Use -p")
		return
	}

	cfg.ReadConf("utilconfig.json")

	if mass {
		for _, t := range cfg.Tasks {
			query_def := strings.Replace(t.Query, "{macr_id}", fmt.Sprint(t.MacrID), 1)
			connect := strings.Replace(t.ConnectString, "{password}", connetion_string, 1)
			CreatePoolCELL(t, connect, query_def)
		}
	} else {
		cfgt := cfg.ReadTask(pool_task)
		query_def := strings.Replace(cfgt.Query, "{macr_id}", fmt.Sprint(cfgt.MacrID), 1)
		connect := strings.Replace(cfgt.ConnectString, "{password}", connetion_string, 1)
		CreatePoolCELL(cfgt, connect, query_def)
	}

}

func CreatePoolCELL(cfgt TasksUtilType, connect string, query_def string) {

	if connect == "" {
		fmt.Println("Connection string not set")
		os.Exit(1)
	}
	db, errdb := sql.Open("oracle", connect)
	if errdb != nil {
		fmt.Println(errdb)
	}

	errdb = db.Ping()
	if errdb != nil {
		fmt.Println(errdb)
		os.Exit(1)
	}

	row, errdb := db.Query(query_def)
	if errdb != nil {
		fmt.Println(errdb)
	}
	defer row.Close()

	file, err := os.OpenFile(cfgt.FileName, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	var (
		lac  string
		cell string
	)

	for row.Next() {
		err := row.Scan(&lac, &cell)
		if err != nil {
			fmt.Println(err)
		}
		_, err = file.WriteString(lac + ";" + cell + "\n")
		if err != nil {
			fmt.Println(err)
		}
	}

	fmt.Println("Util create file " + cfgt.FileName)
}

func (cfg *UtilConf) ReadConf(confname string) {
	file, err := os.Open(confname)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Закрытие при нештатном завершении
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	file.Close()
}

func (cfg *UtilConf) ReadTask(task_name string) TasksUtilType {
	for _, dt := range cfg.Tasks {
		if dt.Name == task_name {
			return dt
		}
	}
	return TasksUtilType{}
}
