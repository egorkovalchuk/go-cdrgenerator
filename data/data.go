package data

import (
	"fmt"
)

type Config struct {
	Common struct {
		Duration  int
		DateRange struct {
			Start string `json:"start"`
			End   string `json:"end"`
			Freq  string `json:"freq"`
		} `json:"date_range"`
		RampUp struct {
			Time  int `json:"time"`
			Steps int `json:"steps"`
		} `json:"ramp_up"`
		OutputDuration int `json:"output_duration"`
	} `json:"common"`
	Tasks []TasksType
}

type TasksType struct {
	Name               string `json:"Name,omitempty"`
	CallsPerSecond     int    `json:"calls_per_second"`
	RecTypeRatio       []RecTypeRatioType
	Percentile         []float64 `json:"percentile"`
	CallsRange         []int     `json:"calls_range"`
	DatapoolCsvFile    string    `json:"datapool_csv_file"`
	PathsToSave        []string  `json:"paths_to_save"`
	Template_save_file string    `json:"template_save_file"`
	CDR_pattern        string    `json:"cdr_pattern"`
}

type RecTypeRatioType struct {
	Name     string `json:"name"`
	Rate     int    `json:"rate"`
	TypeSer  string `json:"type_ser"`
	TypeCodw string `json:"type_code"`
}

type RecTypePool struct {
	Msisdn string
	IMSI   string
}

func HelpStart() {
	fmt.Println("Use -d start deamon mode")
	fmt.Println("Use -s stop deamon mode")
	fmt.Println("Use -t start with debug mode")
}

func CreatePoolList(data [][]string) []RecTypePool {
	var PoolList []RecTypePool
	for i, line := range data {
		if i > 0 { // omit header line
			var rec RecTypePool
			for j, field := range line {
				if j == 0 {
					rec.Msisdn = field
				} else if j == 1 {
					rec.IMSI = field
				}
			}
			PoolList = append(PoolList, rec)
		}
	}
	return PoolList
}
