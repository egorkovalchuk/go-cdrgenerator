package tlv

import (
	"context"
	"io"
	"testing"
	"time"
)

var (
	CamelChannel  = make(chan Camel_tcp, 2000)
	list_listener *ListListener
	camelserver   *Server
	WriteChan     = make(chan WriteStruck, 2000)
)

func StartServerTest(t *testing.T) {

	camel_cfg := &Config{
		Camel_port:       9999,
		Camel_SCP_id:     uint8(Stringtobyte("02")[0]),
		Camel_SMSAddress: "79876543210",
		XVLR:             "79876543210",
		ContryCode:       "250",
		OperatorCode:     "02",
		ResponseFunc:     CamelResponse(),
		RequestFunc:      CamelSend(),
		CamelChannel:     CamelChannel,
	}

	list_listener = NewListListener()
	camelserver = NewServer(camel_cfg, list_listener)

	SetDebug(true)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)

	go camelserver.ServerStart(ctx)
	go CamelWrite(WriteChan)

	// Ждем открытие хотя бы одного соединения
	// Потоки дочерних поднимаются листенером
	for {
		time.Sleep(time.Duration(5) * time.Second)
		if len(camelserver.listeners.List) > 0 {
			break
		}
	}
	cancel()
	<-ctx.Done()
}

func CamelSend() HandReq {
	return func(c *Listener, in chan Camel_tcp) {
		for tmprw := range in {
			// Прописываем id BRT
			tmprw.Frame[0x002C].Param[13] = c.BRTId
			tmp, _ := tmprw.Encoder()
			if _, err = c.WriteTo(tmp); err != nil {
				if err == io.EOF {
					c.Close()
					DeleteCloseConn(c.Server, camelserver)
					return
				}
			}
		}
	}
}

// Обработчик-ответа Camel
func CamelResponse() HandOK {
	return func(c *Listener, camel Camel_tcp) {
	}
}

// Горутина записи в поток Camel
// Эксперимент
func CamelWrite(in chan WriteStruck) {
	for tmp := range in {
		if _, err = tmp.C.WriteTo(tmp.B); err != nil {
			if err == io.EOF {
				tmp.C.Close()
				DeleteCloseConn(tmp.C.Server, camelserver)
			}
		}
	}
}
