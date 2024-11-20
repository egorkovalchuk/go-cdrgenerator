package tlv

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Client struct {
	Address net.IP
}

type Listener struct {
	Server  net.Conn
	Address net.Addr
	BRTId   byte
	mx      sync.RWMutex

	// Эксперимент
	// возможно из-за блокировок может тормозить
	WriteChan chan []byte

	quit chan interface{}
}

// Пишем логи через горутину
type LogStruct struct {
	t    string
	text interface{}
}

// Эксперимент
// Структура для записи в канал
type WriteStruck struct {
	C *Listener
	B []byte
}

// Конфиг Камела
type Config struct {
	Camel_port       int
	Duration         int
	Camel_SCP_id     uint8
	Camel_SMSAddress string
	XVLR             string
	ContryCode       string
	OperatorCode     string
	ResponseFunc     HandOK
	RequestFunc      HandReq
	//Каналы
	CamelChannel chan Camel_tcp
}

type ListListener struct {
	List map[string]*Listener
}

// Возможно сделать как тип Listener (для блокировок)
type HandOK func(*Listener, Camel_tcp)

// Возможно сделать как тип Listener (для блокировок)
// Интерфейс Hand позволяет создавать произвольные объекты
// зарегистрирован для обслуживания определенных сообщений
// должен записать сообщения в Conn и затем вернуться.
// Возврат сигнализирует о том, что запрос завершен и что
// сервер может перейти к следующему запросу по соединению.
type HandReq func(*Listener, chan Camel_tcp)

// Конструктор для открытого соединения
func NewListener(conn net.Conn) *Listener {
	s := &Listener{
		quit: make(chan interface{}),
	}
	s.Server = conn
	s.Address = conn.LocalAddr()
	return s
}

func (p *Listener) WriteTo(tmpwr []byte) (n int, err error) {
	p.mx.Lock()
	n, err = p.Server.Write(tmpwr)
	p.mx.Unlock()
	return
}

func (p *Listener) WriteChannel(in chan []byte) {
	for tmpwr := range in {
		if _, err := p.WriteTo(tmpwr); err != nil {
			LogChannel <- LogStruct{"ERROR", err}
			if err == io.EOF {
				p.Close()
				DeleteCloseConn(p.Server)
				LogChannel <- LogStruct{"INFO", p.RemoteAddr().String() + ": connection close"}
				LogChannel <- LogStruct{"INFO", "Close threads"}
				return
			}
		}
	}
}

func (p *Listener) Read(tmpwr []byte) (n int, err error) {
	p.mx.RLock()
	n, err = p.Server.Read(tmpwr)
	p.mx.RUnlock()
	return
}

func (p *Listener) Close() {
	p.Server.Close()
}

func (p *Listener) SetReadDeadline(t time.Time) (err error) {
	p.mx.Lock()
	err = p.Server.SetReadDeadline(t)
	p.mx.Unlock()
	return
}

func (p *Listener) RemoteAddr() net.Addr {
	t := p.Server.RemoteAddr()
	return t
}

func NewListenerMap() map[string]Listener {
	tmp := make(map[string]Listener)
	return tmp
}

func NewListListener() *ListListener {
	return &ListListener{
		List: make(map[string](*Listener)),
	}
}

func (c *ListListener) SaveOpenConn(value net.Conn) {
	c.List[value.RemoteAddr().String()] = NewListener(value)
	if debug {
		LogChannel <- LogStruct{"DEBUG", "Count CAMEL connection " + fmt.Sprint(len(c.List))}
	}
}

func (c *ListListener) DeleteCloseConn(value net.Conn) {
	delete(c.List, value.RemoteAddr().String())
	if debug {
		LogChannel <- LogStruct{"DEBUG", "Count CAMEL connection " + fmt.Sprint(len(c.List))}
	}
}

func DeleteCloseConn(value net.Conn) {
	list_listener.DeleteCloseConn(value)
}

func (c *ListListener) SaveBRTIdConn(value net.Conn, id byte) {
	tmp := c.List[value.RemoteAddr().String()]
	tmp.BRTId = id
	c.List[value.RemoteAddr().String()] = tmp
}
