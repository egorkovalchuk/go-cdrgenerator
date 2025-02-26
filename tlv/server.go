package tlv

import (
	"context"
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
	LogChannel = make(chan LogStruct, 1000)
	// сделать обнуление
	//Sec uint32 = 0

	// Структура с открытыми соединениями
	//	list_listener *ListListener
	// Конфиг
	//cfg *Config

	// Слушатель открытого порта
	//ln  net.Listener
	err error
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

// Объявление нвого сервера
func NewServer(cfg *Config, listeners *ListListener) *Server {
	s := &Server{
		cfg:       cfg,
		listeners: listeners,
		Sec:       0,
	}
	return s
}

func SetDebug(c bool) {
	debug = c
}

func (s *Server) ServerStart(ctx context.Context) {

	LogChannel <- LogStruct{"INFO", "Starting CAMEL SCP"}
	//list_listener = ll
	//cfg = cfgg
	s.InitMSC()
	// Устанавливаем прослушивание порта

	s.ln, err = net.Listen("tcp", ":"+strconv.Itoa(s.cfg.Camel_port))

	if err != nil {
		LogChannel <- LogStruct{"ERROR", err}
		return
	}
	defer s.ln.Close()

	// Открываем портб занести в цикл для много поточной обработки
	// задать ограничение по количеству открытых коннектов (через структуру)
	// из цикла не работает
	LogChannel <- LogStruct{"INFO", "Start schelduler"}

	for {
		select {
		// Ждем выполнение таймаута
		// Добавить в дальнейшем выход по событию от системы
		case <-ctx.Done():
			s.ln.Close()
			return
		default:
			conn, err := s.ln.Accept()
			if err != nil {
				LogChannel <- LogStruct{"ERROR", err}
				return
			}
			s.listeners.SaveOpenConn(conn)
			// Запуск Обработчика
			go s.CamelHandler(s.listeners.List[conn.RemoteAddr().String()])
			// Запуск Отправки
			// Сделать канал только для этого потока?
			go s.cfg.RequestFunc(s.listeners.List[conn.RemoteAddr().String()], s.cfg.CamelChannel)
			if debug {
				LogChannel <- LogStruct{"DEBUG", "Local address " + conn.LocalAddr().String()}
				LogChannel <- LogStruct{"DEBUG", "Remote address " + conn.RemoteAddr().String()}
			}
		}

	}
	// Выход по таймауту
}

// Один ощий обработчик чтение/запись(keepalive)
func (s *Server) CamelHandler(conn *Listener) {
	defer conn.Close()

	LogChannel <- LogStruct{"INFO", "Client connected from  " + conn.RemoteAddr().String()}
	// KeepAliveServer
	heartbeat := time.NewTicker(20 * time.Second)
	timeoutDuration := 2 * time.Second
	conn.SetReadDeadline(time.Now().Add(timeoutDuration))

	// Буффер обратоки большого количества сообщений
	var buffer_tmp []byte
	cont := 0

	// Запускаем цикл
	for {
		select {
		case <-heartbeat.C:
			// conn.SetReadDeadline(time.Now().Add(timeoutDuration))
			LogChannel <- LogStruct{"INFO", conn.RemoteAddr().String() + ": KeepAlive"}
			// Пока оставляю. но БРТ сам шлет KeepAlive
			// Надо менять запрос
			tmprw := []byte{0, 8, 0, 7, 0, 0, 0, 1}
			if _, err := conn.WriteTo(tmprw); err != nil {
				LogChannel <- LogStruct{"ERROR", err}
				if err == io.EOF {
					LogChannel <- LogStruct{"INFO", conn.RemoteAddr().String() + ": connection close"}
					s.listeners.DeleteCloseConn(conn.Server)
				}
			}
		default:
			// message_io := bufio.NewScanner(conn).Bytes()
			// Буффер это хорошо, но может переполнятся
			// по идее надо вычитывать пакеты с учетом длины
			conn.SetReadDeadline(time.Time{})
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
						s.CamelResponse(conn, camel)
					}
					// считаем пока буффер больше пакета
					if cont < 1 {
						break
					}
				}
			case io.EOF:
				s.listeners.DeleteCloseConn(conn.Server)
				conn.Close()
				LogChannel <- LogStruct{"INFO", conn.RemoteAddr().String() + ": connection close"}
				return
			case os.ErrDeadlineExceeded:
				s.listeners.DeleteCloseConn(conn.Server)
				conn.Close()
				LogChannel <- LogStruct{"INFO", conn.RemoteAddr().String() + ": connection close(ErrDeadlineExceeded1)"}
				return
			default:
				LogChannel <- LogStruct{"ERROR", conn.RemoteAddr().String() + err.Error()}
				//return
				// Сделфать закрытие коннекта  горутины
				//read tcp 127.0.0.1:4868->127.0.0.1:64556: i/o timeout
			}
		}
	}
}

// Основной обработчик входящего трафика
// После стандартных запросов идет переадресация на объявленную функции из основного потока
//func CamelResponse(conn net.Conn, camel Camel_tcp) {
func (s *Server) CamelResponse(conn *Listener, camel Camel_tcp) {
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
		s.listeners.SaveBRTIdConn(conn.Server, camel.Frame[0x0050].Param[0])
	case camel.Type == TYPE_KEEPALIVE_RESP:
		LogChannel <- LogStruct{"INFO", conn.RemoteAddr().String() + ": KeepAlive BRT <- SCP - OK"}
	case camel.Type == TYPE_KEEPALIVE_REQ:
		camel_tmp.Type = TYPE_KEEPALIVE_RESP
		camel_tmp.Sequence = camel.Sequence
		s.Sec = camel.Sequence
		tmprw, _ := camel_tmp.Encoder()
		if _, err = conn.WriteTo(tmprw); err != nil {
			LogChannel <- LogStruct{"ERROR", err}
		}
		LogChannel <- LogStruct{"INFO", conn.RemoteAddr().String() + ": KeepAlive BRT -> SCP"}
	default:
		// Вызов основного обработчика из основного кода
		// Запись ошибок, можно сделать экспериент, с передачей на уровень выше
		s.cfg.ResponseFunc(conn, camel)
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

func (s *Server) ServerStop() {
	LogChannel <- LogStruct{"INFO", "Stoping CAMEL SCP"}
	s.ln.Close()
}
