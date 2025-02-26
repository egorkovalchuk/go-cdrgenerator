# CDR Generator (online/offline)
Generator for creating offline and online load on Credit Control systems (3GPP), billing system testing, analytics, and other applications working with call data records

## Main Features
### Generation of CDR (Call Detail Record) entries in CSV format.
* Configuration of the number of records.
* Support for various call parameters:
* Call start and end time.
* Calling party number (A-number).
* Called party number (B-number).
* Call duration.
* Call result code.
* Ease of use and integration.
* Offline traffic generates CDR files and transfers them to a directory for further processing.
### Online traffic supports:
* Diameter + AVP
* Camel (SCP) (via TVL decoder + ASN)

## Offline Mode
Generates CDR files and transfers them to a specified directory for subsequent processing.
## Online Mode
### Diameter + AVP:
When receiving codes such as **4011** or **Continum** , an offline CDR is generated for Diameter transactions.
### Camel (SCP):
When receiving the code **TYPE_AUTHORIZESMS_REJECT** , an offline CDR is generated for Camel (SCP) transactions.

## Important Note
Camel operates only in daemon mode.
When launching Camel+BRT, sending occurs via Camel (SCP) based on priority.

## Parameters 
* Use **-d** start deamon mode
* Use **-s** stop deamon mode
* Use **-debug** start with debug mode
* Use **-file** save cdr to files(Offline)
* Use **-brt** for test connection to Diameter Credit control systems
* Use **-brtlist** task list (local,roam)
* Use **-camel** for UP SCP Server(Camel protocol)

## Test parameters
* Use -rm Delete all files in directories(Test optional)
* Use -slow_camel to send 1 message every 10 seconds

## Example

```bash
./generator -debug -d -camel -brt -brtlist local
# stop
./generator -s
```