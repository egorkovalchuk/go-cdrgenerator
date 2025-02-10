# CDR Generator (online/offline)
Генератор по созданию оффлайн и онлайн нагрузки на системы Кредитного контроля(3GPP), тестирования систем биллинга, аналитики и других приложений, работающих с данными о звонках

# Основные возможности
## Генерация CDR-записей в формате CSV.
* Настройка количества записей.
* Поддержка различных параметров звонков:
* Время начала и окончания звонка.
* Номер вызывающего абонента (A-номер).
* Номер вызываемого абонента (B-номер).
* Длительность звонка.
* Код результата вызова
* Простота использования и интеграции.

Оффлайн трафик формирует CDR файлы и передает их в каталог для дальнейшей обработки

## Онлайн 
* Diameter + AVP
* Camel(SCP) (через TLV decoder + ASN)

При получении кода **4011** или **Continum** для Diameter формируется офлайн CDR
При получении кода **TYPE_AUTHORIZESMS_REJECT** для CAMEL(SCP) формируется офлайн CDR

# Предупреждение
CAMEL работает только в режиме демона. При запуске CAMEL+BRT -> по приоритет отправки по CAMEL(SCP) 

# Parameters 
* Use **-d** start deamon mode
* Use **-s** stop deamon mode
* Use **-debug** start with debug mode
* Use **-file** save cdr to files(Offline)
* Use **-brt** for test connection to Diameter Credit control systems
* Use **-brtlist** task list (local,roam)
* Use **-camel** for UP SCP Server(Camel protocol)

# Test parameters
* Use -rm Delete all files in directories(Test optional)
* Use -slow_camel to send 1 message every 10 seconds

# Example

```bash
./generator -debug -d -camel -brt -brtlist local
# stop
./generator -s
```