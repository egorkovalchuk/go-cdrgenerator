# Утилита генерации пула LAC/CELL

Этот проект представляет собой утилиту для генерации пула данных LAC (Location Area Code) и CELL (идентификатор соты) на основе конфигурации и запросов к базе данных Oracle.

## Описание

Утилита выполняет следующие задачи:
1. Чтение конфигурации из JSON-файла.
2. Подключение к базе данных Oracle.
3. Выполнение SQL-запросов для получения данных LAC и CELL.
4. Сохранение результатов в CSV-файл.

## Использование

### Флаги командной строки

- `-pool`: Запуск утилиты для создания пула LAC/CELL.
- `-t`: Имя задачи (обязательный параметр).
- `-p`: Пароль для подключения к базе данных (обязательный параметр).
- `-m`: Запуск всех задач из конфигурации.

Пример запуска:
```bash
go run main.go -pool -t task_name -p password
```

## Конфигурация
Конфигурация утилиты задается в файле utilconfig.json. Пример структуры конфигурации:

```json
{
  "Tasks": [
    {
      "Name": "task_name",
      "Macr_id": 1,
      "Region": "region_name",
      "FileName": "output.csv",
      "Query": "SELECT lac, cell FROM table WHERE macr_id = {macr_id}",
      "ConnectString": "user/password@host:port/service"
    }
  ]
}
```

## Зависимости
Используется драйвер github.com/sijms/go-ora/v2 для подключения к Oracle.