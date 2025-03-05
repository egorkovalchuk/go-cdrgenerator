package tlv

import (
	"context"
	"net"
	"testing"
	"time"
)

// MockListener используется для тестирования
type MockListener struct {
	Server net.Conn
}

func (m *MockListener) Close() {
	m.Server.Close()
}

func (m *MockListener) RemoteAddr() net.Addr {
	return m.Server.RemoteAddr()
}

func (m *MockListener) WriteTo(data []byte) (int, error) {
	return m.Server.Write(data)
}

func (m *MockListener) Read(buff []byte) (int, error) {
	return m.Server.Read(buff)
}

func (m *MockListener) SetReadDeadline(t time.Time) error {
	return m.Server.SetReadDeadline(t)
}

// TestServerStart проверяет запуск сервера
func TestServerStart(t *testing.T) {
	cfg := &Config{
		Camel_port: 8080,
		RequestFunc: func(l *Listener, ch chan Camel_tcp) {
			// Mock функция
		},
		ResponseFunc: func(l *Listener, c Camel_tcp) {
			// Mock функция
		},
		CamelChannel: make(chan Camel_tcp),
	}

	server := NewServer(cfg, &ListListener{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем сервер в отдельной горутине
	go server.ServerStart(ctx)

	// Даем серверу время на запуск
	time.Sleep(100 * time.Millisecond)

	// Подключаемся к серверу
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Fatalf("Ошибка при подключении к серверу: %v", err)
	}
	defer conn.Close()

	// Проверяем, что соединение установлено
	if conn == nil {
		t.Error("Соединение не установлено")
	}
}

// TestCamelHandler проверяет обработчик соединений
func TestCamelHandler(t *testing.T) {
	cfg := &Config{
		Camel_port: 8081,
		RequestFunc: func(l *Listener, ch chan Camel_tcp) {
			// Mock функция
		},
		ResponseFunc: func(l *Listener, c Camel_tcp) {
			// Mock функция
		},
		CamelChannel: make(chan Camel_tcp),
	}

	list_listener := NewListListener()
	server := NewServer(cfg, list_listener)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем сервер в отдельной горутине
	go server.ServerStart(ctx)

	// Даем серверу время на запуск
	time.Sleep(100 * time.Millisecond)

	// Подключаемся к серверу
	conn, err := net.Dial("tcp", "localhost:8081")
	if err != nil {
		t.Fatalf("Ошибка при подключении к серверу: %v", err)
	}
	defer conn.Close()

	// Отправляем TLV-сообщение
	message := []byte{0, 8, 0, 7, 0, 0, 0, 1} // Пример KeepAlive
	_, err = conn.Write(message)
	if err != nil {
		t.Fatalf("Ошибка при отправке данных: %v", err)
	}

	// Читаем ответ от сервера
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("Ошибка при чтении ответа: %v", err)
	}

	// Проверяем ответ
	if n == 0 {
		t.Error("Ответ от сервера не получен")
	}
}

// TestServerStop проверяет остановку сервера
func TestServerStop(t *testing.T) {
	cfg := &Config{
		Camel_port: 8082,
		RequestFunc: func(l *Listener, ch chan Camel_tcp) {
			// Mock функция
		},
		ResponseFunc: func(l *Listener, c Camel_tcp) {
			// Mock функция
		},
		CamelChannel: make(chan Camel_tcp),
	}

	server := NewServer(cfg, &ListListener{})
	ctx, cancel := context.WithCancel(context.Background())

	// Запускаем сервер в отдельной горутине
	go server.ServerStart(ctx)

	// Даем серверу время на запуск
	time.Sleep(100 * time.Millisecond)

	// Останавливаем сервер
	cancel()
	server.ServerStop()

	// Проверяем, что сервер остановлен
	_, err := net.Dial("tcp", "localhost:8082")
	if err == nil {
		t.Error("Сервер не остановлен")
	}
}
