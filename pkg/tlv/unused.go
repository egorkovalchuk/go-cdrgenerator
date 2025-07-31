package tlv

import (
	"io"
	"time"
)

// Обработчик только на получение
func (s *Server) CamelHandler1(conn *Listener) {
	defer conn.Close()
	// Буффер обратоки большого количества сообщений
	var buffer_tmp []byte
	cont := 0

	logs.ProcessInfo("Client connected from " + conn.RemoteAddr().String())
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
		case io.EOF:
			s.listeners.DeleteCloseConn(conn.Server)
			conn.Close()
			logs.ProcessInfo(conn.RemoteAddr().String() + ": connection close")
			return
		default:
			logs.ProcessError(conn.RemoteAddr().String() + err.Error())
			//return
			//errors.Is(err, os.ErrDeadlineExceeded)
			//if err, ok := err.(net.Error); ok && err.Timeout()
			//read tcp 127.0.0.1:4868->127.0.0.1:64556: i/o timeout
		}

	}

}
