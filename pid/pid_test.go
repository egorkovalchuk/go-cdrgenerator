package pid

import (
	"fmt"
	"os"
	"testing"
)

const testPIDFile = "test.pid"

func TestSetPID(t *testing.T) {
	err := SetPID(testPIDFile)
	if err != nil {
		t.Fatalf("SetPID failed: %v", err)
	}

	// Проверяем, что файл был создан и содержит PID текущего процесса
	content, err := os.ReadFile(testPIDFile)
	if err != nil {
		t.Fatalf("Failed to read PID file: %v", err)
	}

	pid := os.Getpid()
	expected := fmt.Sprint(pid)
	if string(content) != expected {
		t.Fatalf("PID file content mismatch: expected %s, got %s", expected, string(content))
	}
}

func TestRemovePID(t *testing.T) {
	err := RemovePID(testPIDFile)
	if err != nil {
		t.Fatalf("RemovePID failed: %v", err)
	}

	// Проверяем, что файл был удален
	if _, err := os.Stat(testPIDFile); !os.IsNotExist(err) {
		t.Fatalf("PID file was not removed: %v", err)
	}
}

func TestStopProcess_NonExistentFile(t *testing.T) {
	// Пытаемся остановить процесс с несуществующим PID файлом
	err := StopProcess("non_existent.pid")
	if err != nil {
		t.Fatalf("StopProcess failed: %v", err)
	}
}

func TestStopProcess_InvalidPID(t *testing.T) {
	// Создаем PID файл с невалидным PID
	err := os.WriteFile(testPIDFile, []byte("invalid_pid"), 0666)
	if err != nil {
		t.Fatalf("Failed to create invalid PID file: %v", err)
	}

	// Пытаемся остановить процесс с невалидным PID
	err = StopProcess(testPIDFile)
	if err == nil {
		t.Fatal("Expected error for invalid PID, got nil")
	}

	// Удаляем тестовый файл
	_ = os.Remove(testPIDFile)
}

func TestStopProcess_NonExistentProcess(t *testing.T) {
	// Создаем PID файл с PID несуществующего процесса
	err := os.WriteFile(testPIDFile, []byte("999999"), 0666)
	if err != nil {
		t.Fatalf("Failed to create PID file with non-existent process PID: %v", err)
	}

	// Пытаемся остановить несуществующий процесс
	err = StopProcess(testPIDFile)
	if err != nil {
		t.Fatalf("StopProcess failed: %v", err)
	}

	// Проверяем, что PID файл был удален
	if _, err := os.Stat(testPIDFile); !os.IsNotExist(err) {
		t.Fatalf("PID file was not removed after stopping non-existent process: %v", err)
	}
}

func TestMain(m *testing.M) {
	// Запуск тестов
	code := m.Run()

	// Удаляем тестовый PID файл, если он остался
	_ = os.Remove(testPIDFile)

	os.Exit(code)
}
