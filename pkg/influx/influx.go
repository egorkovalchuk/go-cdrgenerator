package influx

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	InfluxToken   string `json:"InfluxToken"`
	InfluxOrg     string `json:"InfluxOrg"`
	InfluxVersion int    `json:"InfluxVersion"`
	InfluxBucket  string `json:"InfluxBucket"`
	InfluxServer  string `json:"InfluxServer"`
}

// InfluxWriter реализует запись данных в InfluxDB
type InfluxWriter struct {
	config  *Config
	logFunc func(interface{})
}

// NewInfluxWriter создает новый экземпляр
func NewInfluxWriter(cfg *Config, logFunc func(interface{})) *InfluxWriter {
	return &InfluxWriter{
		config:  cfg,
		logFunc: logFunc,
	}
}

// StartHTTPWriter запускает горутину для записи через HTTP
func (w *InfluxWriter) StartHTTPWriter(input <-chan string) {
	go func() {
		request := w.prepareHTTPRequest()

		if request == "" {
			w.logFunc("Stopping influx writer: Unsupported InfluxDB version")
			return
		}

		for str := range input {
			if err := w.sendHTTPRequest(request, str); err != nil {
				w.logFunc(err)
			}
		}
	}()
}

// StartUDPWriter запускает горутину для записи через UDP
func (w *InfluxWriter) StartUDPWriter(input <-chan string) {
	go func() {
		conn, err := w.newUDPClient()
		if err != nil {
			w.logFunc(err)
			return
		}
		defer conn.Close()

		for str := range input {
			if _, err := conn.Write([]byte(str)); err != nil {
				w.logFunc(err)
			}
		}
	}()
}

// Подготовка HTTP-запроса
func (w *InfluxWriter) prepareHTTPRequest() string {
	request := w.config.InfluxServer

	if !strings.HasSuffix(request, "/") {
		request += "/"
	}

	switch w.config.InfluxVersion {
	case 1:
		request += "write?db=" + w.config.InfluxBucket
	case 2:
		request += "api/v2/write?bucket=" + w.config.InfluxBucket +
			"&precision=ns&org=" + w.config.InfluxOrg
	default:
		w.logFunc("Unsupported InfluxDB version")
		return ""
	}

	return request
}

// Отправка HTTP-запроса
func (w *InfluxWriter) sendHTTPRequest(requestURL, data string) error {
	if requestURL == "" {
		return fmt.Errorf("invalid request URL")
	}

	req, err := http.NewRequest("POST", requestURL, nil)
	if err != nil {
		return err
	}

	if w.config.InfluxVersion == 2 {
		if w.config.InfluxToken == "" {
			return fmt.Errorf("influx token is required for v2")
		}
		req.Header.Set("Authorization", "Token "+w.config.InfluxToken)
	}

	req.Header.Set("User-Agent", "go-Influx-Writer")
	req.Body = io.NopCloser(strings.NewReader(data + " " + strconv.FormatInt(time.Now().UnixNano(), 10)))

	client := &http.Client{Timeout: 10 * time.Second}
	defer client.CloseIdleConnections()

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	return nil
}

// Создание UDP-клиента
func (w *InfluxWriter) newUDPClient() (*net.UDPConn, error) {
	address := strings.TrimPrefix(w.config.InfluxServer, "http://")
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	return net.DialUDP("udp", nil, udpAddr)
}
