package data

import (
	"database/sql"
	"encoding/json"
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
	Query         string `json:"Query"`
	ConnectString string `json:"ConnectString"`
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

func CreatePoolCELL(file_name string, confname string, task string, password string) {

	var cfg UtilConf
	cfg.ReadConf(confname)

	cfgt := cfg.ReadTask(task)

	query_def := strings.Replace(cfgt.Query, "{macr_id}", fmt.Sprint(cfgt.MacrID), 1)
	connect := strings.Replace(cfgt.ConnectString, "{password}", password, 1)

	if connect == "" {
		fmt.Println("Connection string not set")
		os.Exit(1)
	}
	db, errdb := sql.Open("oracle", connect)
	if errdb != nil {
		fmt.Println(errdb)
	}

	row, errdb := db.Query(query_def)
	if errdb != nil {
		fmt.Println(errdb)
	}
	defer row.Close()

	file, err := os.OpenFile(file_name, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
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

	fmt.Println("Util create file " + file_name)
}
