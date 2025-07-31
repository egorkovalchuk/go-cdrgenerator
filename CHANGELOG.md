# Change log
## Vesion 0.1
Added offline load
## Vesion 0.2
Added Diameter connection to Nexign NWM produtcs (3GPP Diameter Credit-Control Application)
CCR/CCA type request "Event"
## Vesion 0.2.1
Fix Bug
## Vesion 0.3.0
### Added
Add emulation of the switch operation with a 4011 response (no need credit control)
### Fix Bug
Remove variable in cycle init connection to Diam
## Vesion 0.3.1
### new fearure
Add influx DB write stat
## Vesion 0.4.0
### new fearure
Add CAMEL SCP
## Vesion 0.4.1
### new fearure
The parameter CDR_Pattern has been moved from the array to a higher level

Added monitoring Camel speed

Improved function for working with large buffers revc (SCP server)
### Fix Bug
Fix error when send channel write in files

Fix Diameter send, when no connection was initiated
## Vesion 0.4.2
### new fearure
Added infux DB write stat for Diameter & Camel

Change write log in main process

Added call type generation MO & MT

Removed separate log for Diameter
### Fix Bug
Fix SetReadDeadline timeout for Linux OS
## Vesion 0.4.3
### new fearure
Added offline CDR when got CONFIRM with a Charge 00

Added parametr "thread" - enable start new threads

Added Random MSISDN B

Added -rm optional - remove CDR files in temp directory
## Vesion 0.4.4
### new fearure
Added re-reading of the pool
### Fix Bug
## Vesion 0.4.5
### new fearure
Added Location MSC generate, use pool CELL and LAC area
### Fix Bug
Fix connection termination
## Vesion 0.4.6
### new fearure
Added UDP proto for InfluxDBv1
### Fix Bug
Fix connection to InfluxDBv2
## Vesion 0.5.0
### new fearure
Added  CELL and LAC area pool creation  
## Vesion 0.5.1
### new fearure
### Fix Bug
Fix CAMEL connection

Fix time delay sending
## Vesion 0.5.2
### new fearure
Experimental(one write stream in camel)
### Fix Bug
Fix context 
## Vesion 0.5.2
### new fearure
Added stop for linux
## Vesion 0.5.4
### new fearure
TLV is implemented by the package
### Fix Bug
Fixed crash if there are not enough rights to write offline CDR

Fixed crash if there are incorrect lines in the CSV Pool
## Vesion 0.5.5
### new fearure
InfluxDB is implemented by the package
## Vesion 0.5.6
### new fearure
### Fix Bug
Added handler for closing a network connection from the client side

Added context in tlv connect

Added reconnect Diameter

Added offline CDR generation in the absence of an active connection

Fixed generation of subscriber B

Fixed create new thread
## Vesion 0.5.7
### new fearure
### Fix Bug