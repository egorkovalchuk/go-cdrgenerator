package tlv

import (
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

var (
	debug            bool
	camel_params_map map[uint16]camel_param_desc
	camel_type_map   map[string]camel_type_len

	// Канал записи в лог
	LogChannel = make(chan LogStruct)
	// сделать обнуление
	Sec uint32 = 0

	list_listener *ListListener
	cfg           *Config
)

// Клиент. Не используем, так как являемся сервером
func (c *Client) Dial() error {
	conn, err := net.Dial("tcp", c.Address.String())

	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

func ServerStart(cfgg *Config, ll *ListListener, debugm bool) {
	debug = debugm

	LogChannel <- LogStruct{"INFO", "Starting CAMEL SCP"}
	list_listener = ll
	cfg = cfgg
	// Устанавливаем прослушивание порта

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(cfg.Camel_port))

	if err != nil {
		LogChannel <- LogStruct{"ERROR", err}
		return
	}
	defer ln.Close()

	// Открываем портб занести в цикл для много поточной обработки
	// задать ограничение по количеству открытых коннектов (через структуру)
	// из цикла не работает
	LogChannel <- LogStruct{"INFO", "Start schelduler"}
	heartbeat := time.Tick(time.Duration(cfg.Duration) * time.Second)
	for {
		select {
		// Ждем выполнение таймаута
		// Добавить в дальнейшем выход по событию от системы
		case <-heartbeat:
			ln.Close()
			return
		default:
			conn, err := ln.Accept()
			if err != nil {
				LogChannel <- LogStruct{"ERROR", err}
				return
			}
			ll.SaveOpenConn(conn)
			// Запуск Обработчика
			go CamelHandler(ll.List[conn.LocalAddr().String()])
			// Запуск Отправки
			// Сделать канал только для этого потока?
			go cfg.RequestFunc(ll.List[conn.LocalAddr().String()], cfg.CamelChannel)
		}

	}
	// Выход по таймауту
}

// Обработчик только на получение
func CamelHandler1(conn *Listener) {
	defer conn.Close()
	// Буффер обратоки большого количества сообщений
	var buffer_tmp []byte
	cont := 0

	LogChannel <- LogStruct{"INFO", "Client connected from " + conn.RemoteAddr().String()}
	timeoutDuration := 1 * time.Second
	conn.SetReadDeadline(time.Now().Add(timeoutDuration))

	for {
		conn.SetReadDeadline(time.Now().Add(timeoutDuration))
		buff := make([]byte, 16048)
		n, err := conn.Read(buff)

		switch err {
		case nil:
			message_io := buff[0:n]
			camel := NewCamelTCP()
			buffer_tmp = append(buffer_tmp, message_io...)
			for {
				buffer_tmp, cont, err = camel.DecoderBuffer(buffer_tmp)
				if err != nil {
					LogChannel <- LogStruct{"ERROR", err}
				}
				// если посчитан пакет. то вызываем обработчик
				if cont != -1 {
					CamelResponse(conn, camel)
				}
				// считаем пока буффер больше пакета
				if cont < 1 {
					break
				}
			}
		case io.EOF:
			list_listener.DeleteCloseConn(conn.Server)
			conn.Close()
			LogChannel <- LogStruct{"INFO", conn.RemoteAddr().String() + ": connection close"}
			return
		default:
			//read tcp 127.0.0.1:4868->127.0.0.1:64556: i/o timeout
		}

	}

}

// Один ощий обработчик чтение/запись(keepalive)
func CamelHandler(conn *Listener) {
	defer conn.Close()

	LogChannel <- LogStruct{"INFO", "Client connected from  " + conn.RemoteAddr().String()}
	// KeepAliveServer
	heartbeat := time.Tick(20 * time.Second)
	timeoutDuration := 1 * time.Second
	conn.SetReadDeadline(time.Now().Add(timeoutDuration))

	// Буффер обратоки большого количества сообщений
	var buffer_tmp []byte
	cont := 0

	// Запускаем цикл
	for {
		select {
		case <-heartbeat:
			LogChannel <- LogStruct{"INFO", conn.RemoteAddr().String() + ": KeepAlive"}
			// Пока оставляю. но БРТ сам шлет KeepAlive
			// Надо менять запрос
			tmprw := []byte{0, 8, 0, 7, 0, 0, 0, 1}
			if _, err := conn.WriteTo(tmprw); err != nil {
				LogChannel <- LogStruct{"ERROR", err}
				if err == io.EOF {
					LogChannel <- LogStruct{"INFO", conn.RemoteAddr().String() + ": connection close"}
					list_listener.DeleteCloseConn(conn.Server)
				}
			}
		default:
			//message_io := bufio.NewScanner(conn).Bytes()
			//Буффер это хорошо, но может переполнятся
			//по идее надо вычитывать пакеты с учетом длины
			conn.SetReadDeadline(time.Now().Add(timeoutDuration))
			buff := make([]byte, 16048)
			n, err := conn.Read(buff)

			switch err {
			case nil:
				message_io := buff[0:n]
				camel := NewCamelTCP()
				buffer_tmp = append(buffer_tmp, message_io...)
				for {
					buffer_tmp, cont, err = camel.DecoderBuffer(buffer_tmp)
					if err != nil {
						LogChannel <- LogStruct{"ERROR", err}
					}
					// если посчитан пакет. то вызываем обработчик
					if cont != -1 {
						CamelResponse(conn, camel)
					}
					// считаем пока буффер больше пакета
					if cont < 1 {
						break
					}
				}
			case io.EOF:
				list_listener.DeleteCloseConn(conn.Server)
				conn.Close()
				LogChannel <- LogStruct{"INFO", conn.RemoteAddr().String() + ": connection close"}
				return
			default:
				//read tcp 127.0.0.1:4868->127.0.0.1:64556: i/o timeout
			}
		}
	}
}

// Основной обработчик входящего трафика
// После стандартных запросов идет переадресация на объявленную функции из основного потока
//func CamelResponse(conn net.Conn, camel Camel_tcp) {
func CamelResponse(conn *Listener, camel Camel_tcp) {
	var camel_tmp Camel_tcp
	var err error

	switch {
	case camel.Type == TYPE_STARTUP_REQ:
		camel_tmp.Type = TYPE_STARTUP_RESP
		camel_tmp.Sequence = camel.Sequence
		tmprw, _ := camel_tmp.Encoder()
		if _, err = conn.WriteTo(tmprw); err != nil {
			LogChannel <- LogStruct{"ERROR", err}
		}
		LogChannel <- LogStruct{"INFO", conn.RemoteAddr().String() + ": Initial SCP"}
		list_listener.SaveBRTIdConn(conn.Server, camel.Frame[0x0050].Param[0])
	case camel.Type == TYPE_KEEPALIVE_RESP:
		LogChannel <- LogStruct{"INFO", conn.RemoteAddr().String() + ": KeepAlive BRT <- SCP - OK"}
	case camel.Type == TYPE_KEEPALIVE_REQ:
		camel_tmp.Type = TYPE_KEEPALIVE_RESP
		camel_tmp.Sequence = camel.Sequence
		Sec = camel.Sequence
		tmprw, _ := camel_tmp.Encoder()
		if _, err = conn.WriteTo(tmprw); err != nil {
			LogChannel <- LogStruct{"ERROR", err}
		}
		LogChannel <- LogStruct{"INFO", conn.RemoteAddr().String() + ": KeepAlive BRT -> SCP"}
	default:
		// Вызов основного обработчика из основного кода
		// Запись ошибок, можно сделать экспериент, с передачей на уровень выше
		cfg.ResponseFunc(conn, camel)
	}
}

func init() {
	camel_params_map = make(map[uint16]camel_param_desc)
	for _, i := range camel_params_desc {
		camel_params_map[i.Tag] = i
	}
	camel_type_map = make(map[string]camel_type_len)
	for _, j := range camel_type {
		camel_type_map[j.Type] = j
	}

	go LogWriteForGoRutine(LogChannel)
}

// Запись ошибок из горутин
func LogWriteForGoRutine(text chan LogStruct) {
	var logcamel log.Logger
	filer1, err := os.OpenFile("camel.log", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		logcamel.Fatal(err)
	}

	defer filer1.Close()
	logcamel.SetOutput(filer1)
	logcamel.SetPrefix("")
	logcamel.SetFlags(log.Ldate | log.Ltime)

	for i := range text {
		datetime := time.Now().Local().Format("2006/01/02 15:04:05")
		logcamel.SetPrefix(datetime + " " + i.t + ": ")
		logcamel.SetFlags(0)
		logcamel.Println(i.text)
		logcamel.SetPrefix("")
		logcamel.SetFlags(log.Ldate | log.Ltime)
	}
}
