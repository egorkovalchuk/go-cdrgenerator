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
		return generateRussianMSISDN()
	}

}

// Генерация российского MSISDN
func generateRussianMSISDN() string {
	// Коды российских сотовых операторов
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

// Генерация роумингового MSISDN
func generateRoamingMSISDN() string {
	if opers == nil || len(opers.Operators) == 0 {
		// Если конфиг не загружен, генерируем случайный международный номер
		return generateRandomInternationalMSISDN()
	}

	// Выбираем случайного оператора (не российского)
	var eligibleOperators []Operator
	for _, op := range opers.Operators {
		if op.CountryCode != "7" && len(op.Prefixes) > 0 {
			eligibleOperators = append(eligibleOperators, op)
		}
	}

	if len(eligibleOperators) == 0 {
		return generateRandomInternationalMSISDN()
	}

	// Выбираем случайного оператора
	operatorIndex := rand.Intn(len(eligibleOperators))
	operator := eligibleOperators[operatorIndex]

	// Выбираем случайный префикс оператора
	prefixIndex := rand.Intn(len(operator.Prefixes))
	prefix := operator.Prefixes[prefixIndex]

	// Определяем длину номера
	numberLength := operator.NumberLength
	if numberLength <= len(prefix) {
		numberLength = 10 // дефолтная длина
	}

	// Генерируем оставшиеся цифры
	remainingDigits := numberLength - len(prefix)
	var number string
	for i := 0; i < remainingDigits; i++ {
		digit := rand.Intn(10)
		number += strconv.Itoa(digit)
	}

	// Форматируем номер: код страны + префикс + остальные цифры
	return operator.CountryCode + prefix + number
}

// Генерация случайного международного номера (fallback)
func generateRandomInternationalMSISDN() string {
	// Список популярных кодов стран (без России)
	countryCodes := []string{
		"1",   // США/Канада
		"44",  // Великобритания
		"49",  // Германия
		"33",  // Франция
		"39",  // Италия
		"34",  // Испания
		"81",  // Япония
		"86",  // Китай
		"91",  // Индия
		"61",  // Австралия
		"55",  // Бразилия
		"82",  // Южная Корея
		"90",  // Турция
		"31",  // Нидерланды
		"41",  // Швейцария
		"46",  // Швеция
		"47",  // Норвегия
		"48",  // Польша
		"420", // Чехия
		"36",  // Венгрия
	}

	countryCode := countryCodes[rand.Intn(len(countryCodes))]

	// Генерируем номер длиной 10-12 цифр включая код страны
	totalLength := 10 + rand.Intn(3) // 10-12 цифр
	numberLength := totalLength - len(countryCode)

	if numberLength <= 0 {
		numberLength = 7
	}

	var number string
	// Первая цифра после кода страны не должна быть 0
	number += strconv.Itoa(rand.Intn(9) + 1)

	// Остальные цифры
	for i := 1; i < numberLength; i++ {
		digit := rand.Intn(10)
		number += strconv.Itoa(digit)
	}

	return countryCode + number
}

// Дополнительные вспомогательные функции

// GetRandomOperator возвращает случайного оператора для роуминга
func GetRandomOperator() *Operator {
	if opers == nil || len(opers.Operators) == 0 {
		return nil
	}

	var internationalOperators []Operator
	for _, op := range opers.Operators {
		if op.CountryCode != "7" {
			internationalOperators = append(internationalOperators, op)
		}
	}

	if len(internationalOperators) == 0 {
		return nil
	}

	operatorIndex := rand.Intn(len(internationalOperators))
	return &internationalOperators[operatorIndex]
}

// GenerateMSISDNForOperator генерирует номер для конкретного оператора
func GenerateMSISDNForOperator(operator Operator) string {
	if len(operator.Prefixes) == 0 {
		return ""
	}

	prefixIndex := rand.Intn(len(operator.Prefixes))
	prefix := operator.Prefixes[prefixIndex]

	numberLength := operator.NumberLength
	if numberLength <= len(prefix) {
		numberLength = 10
	}

	remainingDigits := numberLength - len(prefix)
	var number string
	for i := 0; i < remainingDigits; i++ {
		digit := rand.Intn(10)
		number += strconv.Itoa(digit)
	}

	return operator.CountryCode + prefix + number
}
