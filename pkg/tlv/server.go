package tlv

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/egorkovalchuk/go-cdrgenerator/pkg/logger"
)

var (
	debug            bool
	camel_params_map map[uint16]camel_param_desc
	camel_type_map   map[string]camel_type_len

	loggerOnce sync.Once
	logs       *logger.LogWriter
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

// NewServer создает новый экземпляр сервера.
func NewServer(cfg *Config, listeners *ListListener) *Server {
	s := &Server{
		cfg:       cfg,
		listeners: listeners,
		Sec:       0,
	}
	return s
}

// SetDebug устанавливает режим отладки.
func SetDebug(c bool) {
	debug = c
	logs.ChangeDebugLevel(debug)
}

func (s *Server) ServerStart(ctx context.Context) {

	logs.ProcessInfo("Starting CAMEL SCP")
	Init()
	s.InitMSC()

	var err error
	// Устанавливаем прослушивание порта
	s.ln, err = net.Listen("tcp", ":"+strconv.Itoa(s.cfg.Camel_port))

	if err != nil {
		logs.ProcessError(err)
		return
	}
	defer s.ln.Close()

	// Открываем портб занести в цикл для много поточной обработки
	// задать ограничение по количеству открытых коннектов (через структуру)
	// из цикла не работает
	logs.ProcessInfo("Start schelduler")

	for {
		select {
		// Ждем выполнение таймаута
		// Добавить в дальнейшем выход по событию от системы
		case <-ctx.Done():
			return
		default:
			conn, err := s.ln.Accept()
			if err != nil {
				logs.ProcessError(err)
				return
			}
			s.listeners.SaveOpenConn(conn, ctx)
			// Запуск Обработчика
			go s.CamelHandler(s.listeners.List[conn.RemoteAddr().String()])
			// Запуск Отправки
			// Запуск потока только после инициализации перенесено в CamelResponse
			// go s.cfg.RequestFunc(s.listeners.List[conn.RemoteAddr().String()], s.cfg.CamelChannel)
			logs.ProcessDebug("Local address " + conn.LocalAddr().String())
			logs.ProcessDebug("Remote address " + conn.RemoteAddr().String())
		}

	}
	// Выход по таймауту
}

// Один ощий обработчик чтение/запись(keepalive)
func (s *Server) CamelHandler(conn *Listener) {
	defer conn.Close()

	logs.ProcessInfo("Client connected from  " + conn.RemoteAddr().String())
	// KeepAliveServer
	heartbeat := time.NewTicker(20 * time.Second)
	timeoutDuration := 2 * time.Second
	conn.SetReadDeadline(time.Now().Add(timeoutDuration))

	// Буффер обратоки большого количества сообщений
	var buffer_tmp []byte
	var err error
	// cont := 0

	// Запускаем цикл
	for {
		select {
		case <-heartbeat.C:
			s.sendKeepAlive(conn)
		case <-conn.Ctx.Done():
			return
		default:
			conn.SetReadDeadline(time.Now().Add(timeoutDuration))
			// conn.SetReadDeadline(time.Time{})
			buffer_tmp, err = s.readNetConn(conn, buffer_tmp)
			// err = s.readNetConnLenght(conn)
			if err != nil {
				return
			}
		}
	}
}

// Функция чтения из буфера
func (s *Server) readNetConn(conn *Listener, buffer_tmp []byte) ([]byte, error) {
	// message_io := bufio.NewScanner(conn).Bytes()
	// Буффер это хорошо, но может переполнятся
	// по идее надо вычитывать пакеты с учетом длины
	cont := 0
	buff := make([]byte, 16048)
	n, err := conn.Read(buff)

	if err != nil {
		return buffer_tmp, s.handleReadError(conn, err)
	}

	camel := NewCamelTCP()
	buffer_tmp = append(buffer_tmp, buff[:n]...)

	for {
		buffer_tmp, cont, err = camel.DecoderBuffer(buffer_tmp)
		if err != nil {
			logs.ProcessError(err)
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

	return buffer_tmp, nil
}

// handleReadError обрабатывает ошибки чтения.
func (s *Server) handleReadError(conn *Listener, err error) error {
	switch err {
	case io.EOF:
		conn.Close()
		s.listeners.DeleteCloseConn(conn.Server)
		logs.ProcessInfo(conn.RemoteAddr().String() + ": connection close")
		return err
	case os.ErrDeadlineExceeded:
		conn.Close()
		s.listeners.DeleteCloseConn(conn.Server)
		logs.ProcessInfo(conn.RemoteAddr().String() + ": connection close (ErrDeadlineExceeded)")
		return err
	case net.ErrClosed:
		conn.Close()
		s.listeners.DeleteCloseConn(conn.Server)
		logs.ProcessInfo(conn.RemoteAddr().String() + ": connection close (net.ErrClosed)")
		return err
	default:
		logs.ProcessError(conn.RemoteAddr().String() + ": " + err.Error())
	}
	return nil
}

// sendKeepAlive отправляет KeepAlive-сообщение.
func (s *Server) sendKeepAlive(conn *Listener) {
	logs.ProcessInfo(conn.RemoteAddr().String() + ": KeepAlive")
	// Пока оставляю. но БРТ сам шлет KeepAlive
	// Надо менять запрос
	tmprw := []byte{0, 8, 0, 7, 0, 0, 0, 1}
	if _, err := conn.WriteTo(tmprw); err != nil {
		logs.ProcessError(err)
		if err == io.EOF {
			s.listeners.DeleteCloseConn(conn.Server)
			logs.ProcessInfo(conn.RemoteAddr().String() + ": connection close")
		}
	}
}

// Основной обработчик входящего трафика
// После стандартных запросов идет переадресация на объявленную функции из основного потока
// func CamelResponse(conn net.Conn, camel Camel_tcp) {
func (s *Server) CamelResponse(conn *Listener, camel Camel_tcp) {
	var camel_tmp Camel_tcp
	var err error

	switch camel.Type {
	case TYPE_STARTUP_REQ:
		camel_tmp.Type = TYPE_STARTUP_RESP
		camel_tmp.Sequence = camel.Sequence
		tmprw, _ := camel_tmp.Encoder()
		if _, err = conn.WriteTo(tmprw); err != nil {
			logs.ProcessError(err)
		}
		logs.ProcessInfo(conn.RemoteAddr().String() + ": Initial SCP")
		s.listeners.SaveBRTIdConn(conn.Server, camel.Frame[0x0050].Param[0])
		// Запуск Отправки
		// Запуск потока только после инициализации
		logs.ProcessInfo(conn.RemoteAddr().String() + ": Starting a data transfer stream")
		go s.cfg.RequestFunc(s.listeners.List[conn.RemoteAddr().String()], s.cfg.CamelChannel)
	case TYPE_KEEPALIVE_RESP:
		logs.ProcessInfo(conn.RemoteAddr().String() + ": KeepAlive BRT <- SCP - OK")
	case TYPE_KEEPALIVE_REQ:
		camel_tmp.Type = TYPE_KEEPALIVE_RESP
		camel_tmp.Sequence = camel.Sequence
		s.Sec = camel.Sequence
		tmprw, _ := camel_tmp.Encoder()
		if _, err = conn.WriteTo(tmprw); err != nil {
			logs.ProcessError(err)
		}
		logs.ProcessInfo(conn.RemoteAddr().String() + ": KeepAlive BRT -> SCP")
	case TYPE_SHUTDOWN_REQ:
		camel_tmp.Type = TYPE_SHUTDOWN_RESP
		camel_tmp.Sequence = camel.Sequence
		tmp := NewCamelTCPParam()
		tmp.Tag = camel_params_map[0x0013].Tag
		tmp.Param = []byte{0, 0, 0, 0}
		tmp.LengthParams = uint16(camel_params_map[0x0013].MaxLen)
		tmp.Type = camel_params_map[tmp.Tag].Type
		camel_tmp.Frame[tmp.Tag] = tmp
		tmprw, _ := camel_tmp.Encoder()
		if _, err = conn.WriteTo(tmprw); err != nil {
			logs.ProcessError(err)
		}
		conn.Close()
		s.listeners.DeleteCloseConn(conn.Server)
		logs.ProcessInfo(conn.RemoteAddr().String() + ": connection close")
	default:
		// Вызов основного обработчика из основного кода
		// Запись ошибок, можно сделать экспериент, с передачей на уровень выше
		s.cfg.ResponseFunc(conn, camel)
	}
}

// init инициализирует глобальные переменные.
func Init() {
	camel_params_map = make(map[uint16]camel_param_desc)
	for _, i := range camel_params_desc {
		camel_params_map[i.Tag] = i
	}
	camel_type_map = make(map[string]camel_type_len)
	for _, j := range camel_type {
		camel_type_map[j.Type] = j
	}

	loggerOnce.Do(func() {
		logs = logger.NewLogWriter("camel.log", debug)
		go logs.LogWriteForGoRutineStruct()
	})
}

func (s *Server) ServerStop() {
	logs.ProcessInfo("Stoping CAMEL SCP")
	s.ln.Close()
}

// Функция чтения из буфера
// Эксперимент
func (s *Server) readNetConnLenght(conn *Listener) error {
	header := make([]byte, 2)
	_, err := io.ReadFull(conn.Server, header)
	if err != nil {
		return s.handleReadError(conn, err)
	}

	length := binary.BigEndian.Uint16(header)

	value := make([]byte, length)
	_, err = io.ReadFull(conn.Server, value)
	if err != nil {
		return s.handleReadError(conn, err)
	}

	camel := NewCamelTCP()
	_, _, err = camel.DecoderBuffer(value)
	if err != nil {
		logs.ProcessError(err)
	}
	s.CamelResponse(conn, camel)
	return nil
}
