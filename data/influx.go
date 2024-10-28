package data

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func StartWriteInflux(cfg CommonType, f func(logtext interface{}), InputString chan string) {
	heartbeat := time.Tick(10 * time.Second)

	for {
		select {
		case <-heartbeat:
			f("OK")

		case str := <-InputString:
			request := cfg.Report.InfluxServer

			resp, err := http.NewRequest("POST", request, nil)
			if err != nil {
				f(err)
			}

			if cfg.Report.LoginInflux != "" {
				resp.SetBasicAuth(cfg.Report.LoginInflux, cfg.Report.PassInflux)
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
		default:
		}
	}
}
