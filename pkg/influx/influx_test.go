package influx

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockLogger используется для захвата логов в тестах
type MockLogger struct {
	Messages []string
}

func (m *MockLogger) Log(message interface{}) {
	m.Messages = append(m.Messages, fmt.Sprint(message))
}

// TestNewInfluxWriter проверяет создание нового InfluxWriter
func TestNewInfluxWriter(t *testing.T) {
	cfg := &Config{
		InfluxToken:   "test-token",
		InfluxOrg:     "test-org",
		InfluxVersion: 2,
		InfluxBucket:  "test-bucket",
		InfluxServer:  "http://localhost:8086",
	}

	logger := &MockLogger{}
	writer := NewInfluxWriter(cfg, logger.Log)

	if writer == nil {
		t.Error("Expected a non-nil InfluxWriter, got nil")
	}
}

// TestPrepareHTTPRequest проверяет формирование HTTP-запроса
func TestPrepareHTTPRequest(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "InfluxDB v1",
			config: Config{
				InfluxVersion: 1,
				InfluxBucket:  "test-bucket",
				InfluxServer:  "http://localhost:8086",
			},
			expected: "http://localhost:8086/write?db=test-bucket",
		},
		{
			name: "InfluxDB v2",
			config: Config{
				InfluxVersion: 2,
				InfluxBucket:  "test-bucket",
				InfluxOrg:     "test-org",
				InfluxServer:  "http://localhost:8086",
			},
			expected: "http://localhost:8086/api/v2/write?bucket=test-bucket&precision=ns&org=test-org",
		},
		{
			name: "Unsupported version",
			config: Config{
				InfluxVersion: 3,
				InfluxServer:  "http://localhost:8086",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &MockLogger{}
			writer := NewInfluxWriter(&tt.config, logger.Log)
			request := writer.prepareHTTPRequest()

			if request != tt.expected {
				t.Errorf("Expected request URL %q, got %q", tt.expected, request)
			}
		})
	}
}

// TestSendHTTPRequest проверяет отправку HTTP-запроса
func TestSendHTTPRequest(t *testing.T) {
	// Создаем mock HTTP-сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Token test-token" {
			t.Errorf("Expected Authorization header 'Token test-token', got %q", authHeader)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := &Config{
		InfluxToken:   "test-token",
		InfluxOrg:     "test-org",
		InfluxVersion: 2,
		InfluxBucket:  "test-bucket",
		InfluxServer:  server.URL,
	}

	logger := &MockLogger{}
	writer := NewInfluxWriter(cfg, logger.Log)

	data := "cpu_load,host=server01 value=0.64"
	err := writer.sendHTTPRequest(server.URL, data)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// TestNewUDPClient проверяет создание UDP-клиента
func TestNewUDPClient(t *testing.T) {
	// Запускаем mock UDP-сервер
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	cfg := &Config{
		InfluxServer: "http://" + conn.LocalAddr().String(),
	}

	logger := &MockLogger{}
	writer := NewInfluxWriter(cfg, logger.Log)

	udpConn, err := writer.newUDPClient()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	defer udpConn.Close()

	if udpConn == nil {
		t.Error("Expected a non-nil UDP connection, got nil")
	}
}

// TestStartUDPWriter проверяет запись данных через UDP
func TestStartUDPWriter(t *testing.T) {
	// Запускаем mock UDP-сервер
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	cfg := &Config{
		InfluxServer: "http://" + conn.LocalAddr().String(),
	}

	logger := &MockLogger{}
	writer := NewInfluxWriter(cfg, logger.Log)

	input := make(chan string, 1)
	defer close(input)

	go writer.StartUDPWriter(input)

	data := "cpu_load,host=server01 value=0.64"
	input <- data

	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		t.Fatal(err)
	}

	received := string(buffer[:n])
	if received != data {
		t.Errorf("Expected data %q, got %q", data, received)
	}
}
