# Формат передачи номеров Имя  параметра содержит окончание Information.
## Пример:
CalledPartyNumber - Цифры номера вызываемого абонента
CalledPartyNumberInformation - Параметры номера вызываемого абонента
Параметр NumberInformation имеет размер от 1 до 3 байт.
1 байт – значение NPI
2 байт – значение TON (если есть)
3 байт – значение Screening
### Значения поля NPI
Unknown 00000000
ISDN (E163/E164) 00000001
Land Mobile (E.212) 00000110
ISDN Mobile (E.214) 00000111
### Значения поля TON:
TON Значение
Unknown 00000000
International 00000001
National 00000010
Network Specific 00000011
Subscriber Number 00000100
Alphanumeric 00000101
Abbreviated 00000110

# Значения поля SMSStatus
Submitted 0x00
systemFailure 0x01
unexpectedDataValue 0x02
facilityNotSupported 0x03
sM-DeliveryFailure 0x04
releaseFromRadioInterface 0x05
unknownFailure 0xFF

# Значения поля EndReason
OK 0
Abort 1
Timeout 2
Invalid Encoding 3
Reserved 4..255