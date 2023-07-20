package tlv

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

// является сервером при работе через
// надо изучить логику
type Client struct {
	Address net.IP
}

type Server struct {
	Address net.Resolver
}

// Клиент. Не используем, так как являемся сервером
func (c *Client) Dial() error {
	conn, err := net.Dial("tcp", c.Address.String())

	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

func (c *Server) Server() {

	// Устанавливаем прослушивание порта
	ln, _ := net.Listen("tcp", ":8081")

	// Открываем порт
	conn, _ := ln.Accept()

	// Запускаем цикл
	for {
		// Будем прослушивать все сообщения разделенные \n
		message, _ := bufio.NewReader(conn).ReadString('\n')
		// Распечатываем полученое сообщение
		fmt.Print("Message Received:", string(message))
		// Процесс выборки для полученной строки
		newmessage := strings.ToUpper(message)
		// Отправить новую строку обратно клиенту
		conn.Write([]byte(newmessage + "\n"))
	}

}
