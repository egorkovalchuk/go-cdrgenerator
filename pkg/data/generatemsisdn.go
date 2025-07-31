package data

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
)

type Operator struct {
	Name         string   `json:"name"`
	Country      string   `json:"country"`
	CountryCode  string   `json:"countryCode"`
	Prefixes     []string `json:"prefixes"`
	NumberLength int      `json:"numberLength"`
	LACRange     [2]int   `json:"lacRange"`
	CellIDRange  [2]int   `json:"cellIdRange"`
}

type Operators struct {
	Operators []Operator `json:"operators"`
}

var (
	opers *Operators
)

func init() {
	var err error
	opers, err = loadConfig("operators.json")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
}

func loadConfig(filename string) (*Operators, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var opers Operators
	if err := json.Unmarshal(data, &opers); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &opers, nil
}

// Генерация номера
func RandomMSISDN(tsk string) string {
	if tsk == "roam" {
		// логика для международных
		return ""
	} else {
		// Коды российских сотовых операторов (2023)
		mobilePrefixes := []string{
			"901", "902", "903", "904", "905", "906", "908", "909", // Tele2
			"915", "916", "917", "919", "985", "986", "987", "989", // МТС
			"921", "922", "923", "924", "925", "926", "927", "928", "929", "931", // Мегафон
			"930", "931", "932", "933", "934", "936", "937", "938", "939", // Билайн
			"950", "951", "952", "953", "954", "955", "956", "958", // Yota
			"960", "961", "962", "963", "964", "965", "966", "967", "968", "969", // другие операторы
			"970", "971", "972", "973", "974", "975", "976", "977", "978", "979",
			"980", "981", "982", "983", "984", "988",
		}
		// Выбираем случайный префикс
		prefixIndex := rand.Intn(len(mobilePrefixes))
		prefix := mobilePrefixes[prefixIndex]
		// Генерируем остальные 7 цифр номера
		var number string
		for i := 0; i < 7; i++ {
			digit := rand.Intn(10)
			number += strconv.Itoa(digit)
		}

		// Форматируем номер в международном формате
		return "7" + prefix + number
	}

}
