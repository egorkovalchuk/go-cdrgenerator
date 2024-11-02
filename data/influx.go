package data

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Запуск горутины записи статистики в БД
func StartWriteInflux(cfg CommonType, f func(logtext interface{}), InputString chan string) {

	request := cfg.Report.InfluxServer

	if !strings.HasSuffix(request, "/") {
		request += "/"
	}

	switch cfg.Report.InfluxVersion {
	case 1:
		request += "write?db=" + cfg.Report.InfluxBucket
	case 2:
		request += "api/v2/write?bucket=" + cfg.Report.InfluxBucket + "&precision=ns&org=" + cfg.Report.InfluxOrg
	default:
		f("Database version not specified")
		return
	}

	for str := range InputString {

		resp, err := http.NewRequest("POST", request, nil)

		if err != nil {
			f(err)
		}

		if cfg.Report.InfluxVersion == 2 {
			if cfg.Report.InfluxToken != "" {
				resp.Header.Set("Authorization", "Token "+cfg.Report.InfluxToken)
			} else {
				f("Stopping InfluxDB write, Token is empty")
			}
		}
		//resp.Header.Add("Content-Type", "application/json")
		resp.Header.Add("User-Agent", "go-LT-Report")
		resp.Body = ioutil.NopCloser(strings.NewReader(str + " " + fmt.Sprintf("%d", time.Now().UnixNano())))

		cli := &http.Client{}
		rsp, err := cli.Do(resp)

		if err != nil {
			f(err)
		}

		if rsp.StatusCode <= 200 || rsp.StatusCode >= 299 {
			f("Write error " + strconv.Itoa(rsp.StatusCode) + " " + request + str)
		}

		defer cli.CloseIdleConnections()
	}
}

func StartWriteInfluxUDPV1(cfg CommonType, f func(logtext interface{}), InputString chan string) {

	conn, err := NewUDPClient(cfg)
	if err != nil {
		f(err)
	}

	defer conn.Close()

	for str := range InputString {
		// В каком формате UDP?
		_, err := conn.Write([]byte(str))
		if err != nil {
			f(err)
		}
		f(str)
	}
}

func NewUDPClient(cfg CommonType) (*net.UDPConn, error) {
	var udpAddr *net.UDPAddr
	var url string
	if strings.HasPrefix(cfg.Report.InfluxServer, "http://") {
		url = cfg.Report.InfluxServer[len("http://"):]
	} else {
		url = cfg.Report.InfluxServer
	}

	udpAddr, err := net.ResolveUDPAddr("udp", url)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}

	return conn, err
}
