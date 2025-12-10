# CDR Generator (online/offline)
Генератор тестовых CDR (Call Detail Records) для нагрузочного тестирования систем Кредитного контроля телекоммуникационной экосистемы. Поддерживает как офлайн-генерацию CDR файлов, так и онлайн-взаимодействие с системами биллинга по протоколам Diameter и CAMEL.

## Основные возможности
### Оффлайн режим
- **Генерация CDR записей** в формате CSV
- **Гибкая настройка** количества записей и параметров вызовов
- **Конфигурируемые параметры вызовов**:
  - Время начала и окончания звонка
  - Номер вызывающего абонента (A-номер)
  - Номер вызываемого абонента (B-номер)
  - Продолжительность звонка
  - Код результата вызова
  - Тип вызова (голос/SMS)
  - LAC/CELL информация
- **Распределенная запись** в несколько директорий
- **Автоматическая ротация** файлов

Оффлайн трафик формирует CDR файлы и передает их в каталог для дальнейшей обработки

### Онлайн режим
#### Поддержка протокола Diameter (RFC 4006)
- **Credit-Control** запросы (CCR)
- **Поддержка AVP** (Attribute-Value Pairs)
- **Автоматический переход в офлайн** при кодах ответа 4011, 4522, 4012
- **Переподключение** при разрыве соединения
- **Watchdog-механизм** (DWR/DWA)

При получении кода **4011** или **Continum** для Diameter формируется офлайн CDR

#### Поддержка протокола CAMEL (SCP(через TLV decoder + ASN))
- **Сервер SCP** для обработки CAMEL-сообщений
- **Поддержка типов сообщений**:
  - `TYPE_AUTHORIZESMS_REJECT/CONFIRM`
  - `TYPE_AUTHORIZEVOICE_REJECT/CONFIRM`
  - `TYPE_ENDSMS_RESP`
  - `TYPE_ENDVOICE_RESP`
- **TVL-декодирование** + ASN.1
- **Приоритетная маршрутизация** CAMEL+BRT

При получении кода **TYPE_AUTHORIZESMS_REJECT** для CAMEL(SCP) формируется офлайн CDR

## Архитектура

### Компоненты
1. **Генератор нагрузки** - создает тестовые вызовы на основе CSV-пулов абонентов
2. **Менеджер потоков** - управляет worker-горутинами для параллельной обработки
3. **Диаметр-клиент** - взаимодействие с BRT (Balance and Rating Tool)
4. **CAMEL-сервер** - обработка online-CDR через SCP
5. **Мониторинг** - сбор статистики и метрик (поддержка InfluxDB)
6. **Логирование** - структурированное логирование с ротацией

### Ключевые особенности
- **Многопоточность** - настраиваемое количество worker-горутин
- **Балансировка нагрузки** - автоматическое создание дополнительных потоков при падении скорости
- **Статистика в реальном времени** - мониторинг скорости и кодов ответов
- **Поддержка пулов данных** - CSV-файлы с абонентами и LAC/CELL информацией
- **Конфигурация через JSON** - гибкая настройка задач и параметров

## Установка и запуск

### Требования
- Go 1.21 или выше
- Доступ к BRT-серверам (для онлайн-режима)
- CSV-файлы с данными абонентов

### Сборка
```bash
git clone https://github.com/egorkovalchuk/go-cdrgenerator.git
cd go-cdrgenerator
go build -o generator cmd/generator.go
```

### Конфигурация
Перед запуском настройте `config.json`:
```json
{
  "Common": {
    "Duration": 14400,
    "BRT": ["192.168.1.100"],
    "BRT_port": 3868,
    "BRT_OriginHost": "generator.example.com",
    "BRT_OriginRealm": "example.com",
    "CAMEL": {
      "Port": 12345,
      "Camel_SCP_id": "1",
      "SMSCAddress": "79161234567",
      "XVLR": "79161234567"
    },
    "Report": {
      "Influx": false,
      "Region": "test"
    }
  },
  "Tasks": [
    {
      "Name": "local",
      "CallsPerSecond": 100,
      "DatapoolCsvFile": "pool/local.csv",
      "DatapoolCsvLac": "pool/lac.csv",
      "DefaultMSISDN_B": "79001234567",
      "DefaultLAC": 1001,
      "DefaultCELL": 101,
      "PathsToSave": ["/var/cdr/local/"],
      "Template_save_file": "cdr_{date}.cdr",
      "CDR_pattern": "default",
      "RecTypeRatio": [
        {"Name": "voice_local", "Record_type": "01", "TypeService": "1", "Rate": 70},
        {"Name": "sms_local", "Record_type": "09", "TypeService": "17", "Rate": 30}
      ]
    }
  ]
}
```
## Использование

### Параметры командной строки

#### Основные параметры:
```
-config string     Файл конфигурации (по умолчанию "config.json")
-debug             Режим отладки
-d                 Запуск в режиме демона (daemon mode)
-s                 Остановка демона
-v                 Вывод версии
```

#### Режимы работы:
```
-brt               Подключение к BRT по протоколу Diameter
-camel             Запуск SCP-сервера для протокола CAMEL
-file              Запись CDR в файлы (офлайн режим)
```

#### Параметры BRT:
```
-brtlist value     Список задач для работы с BRT (например: "local,roam")
                   Доступные значения: local, roam, и другие из config.json
```

#### Тестовые параметры:
```
-rm                Удалить все файлы в директориях после работы (тест)
-slow              Равномерная отправка сообщений с задержками
-slow_camel        Отправка CAMEL-сообщений раз в 10 секунд
-thread            Разрешить запуск дополнительных потоков
```

### Примеры использования

#### 1. Офлайн генерация CDR файлов:
```bash
./generator -debug -file -config config.json
```

#### 2. Онлайн режим с Diameter:
```bash
./generator -debug -d -brt -brtlist local,roam -config config.json
```

#### 3. Онлайн режим с CAMEL:
```bash
./generator -debug -d -camel -config config.json
```

#### 4. Смешанный режим (Diameter + CAMEL):
```bash
./generator -debug -d -camel -brt -brtlist local -config config.json
```

#### 5. Остановка демона:
```bash
./generator -s
```

#### 6. Тестовый режим с очисткой:
```bash
./generator -debug -file -rm -config config.json
```

#### Предупреждение
CAMEL работает только в режиме демона. При запуске CAMEL+BRT -> по приоритет отправки по CAMEL(SCP) 

## Формат данных

### CSV пул абонентов (пример):
```csv
MSISDN;IMSI;CallsCount
79161234567;250012345678901;10
79167654321;250098765432109;5
```

### Формат CDR записи:
```
ID;Caller;Callee;StartTime;EndTime;Duration;CallType;Result;Termination;LAC;CELL
550e8400-e29b-41d4-a716-446655440000;79161234567;79001234567;2024-01-15T10:30:00Z;2024-01-15T10:30:45Z;45;01;0;normal;1001;101
```

### LAC/CELL пул (пример):
```csv
LAC;CELL
1001;101
1001;102
1002;201
```

## Мониторинг и статистика

### Метрики в реальном времени:
- **Скорость генерации** (операций/секунду)
- **Количество отправленных/полученных сообщений**
- **Коды ответов Diameter/CAMEL**
- **Статистика по типам вызовов**

### Интеграция с InfluxDB:
```json
"Report": {
  "Influx": true,
  "InfluxServer": "http://localhost:8086",
  "InfluxToken": "your-token",
  "InfluxOrg": "your-org",
  "InfluxBucket": "cdr-metrics",
  "InfluxVersion": "2",
  "Region": "production"
}
```

## Особенности работы

### Приоритеты обработки:
1. **CAMEL** (если подключены CAMEL-клиенты)
2. **Diameter** (если подключены BRT-сервера и задача в brtlist)
3. **Файлы** (оффлайн режим)

### Обработка ошибок:
- **Diameter**: Коды 4011, 4522, 4012 → генерация offline-CDR
- **CAMEL**: `TYPE_AUTHORIZESMS_REJECT` → генерация offline-CDR
- **Неизвестные абоненты** (5030) → пропуск записи

### Управление памятью:
- **Пул абонентов** загружается в память
- **Офлайн-CDR кэш** для ожидающих ответов
- **Каналы (channels)** для межгорутинной коммуникации

## Совместимость

### Поддерживаемые протоколы:
- **Diameter** (RFC 4006, 3588)
- **CAMEL Phase 3+** (ETSI TS 129 078)

### Тестировано с:
- Системами биллинга операторов связи
- BRT (Balance and Rating Tool)
- SCP (Service Control Point) серверами

## Безопасность

- **PID файлы** для управления демоном
- **Проверка прав доступа** к директориям записи
- **Контексты** для graceful shutdown
- **Логирование** всех операций

## Разработка

### Структура проекта:
```
go-cdrgenerator/
├── cmd/
│   └── generator.go          # Основной исполняемый файл
├── pkg/
│   ├── data/                # Структуры данных и утилиты
│   ├── diameter/            # Diameter-клиент
│   ├── tlv/                 # CAMEL (TLV) обработка
│   ├── logger/              # Логирование
│   ├── influx/              # InfluxDB интеграция
│   └── pid/                 # Управление PID-файлами
├── config.json.example      # Пример конфигурации
└── README.md               # Документация
```

### Зависимости:
- `github.com/fiorix/go-diameter/v4` - Diameter протокол
- `github.com/egorkovalchuk/go-cdrgenerator/pkg/*` - внутренние пакеты

## Лицензия

MIT License.

## Автор

Egor Kovalchuk

## Версия

v0.5.8



