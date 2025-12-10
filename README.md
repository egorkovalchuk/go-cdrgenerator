# CDR Generator (online/offline)

Generator for test CDR (Call Detail Records) for load testing of Credit Control systems in telecommunications ecosystem. Supports both offline CDR file generation and online interaction with billing systems via Diameter and CAMEL protocols.

## Main Features

### Offline Mode
- **CDR record generation** in CSV format
- **Flexible configuration** of record count and call parameters
- **Configurable call parameters**:
  - Call start and end time
  - Calling party number (A-number)
  - Called party number (B-number)
  - Call duration
  - Call result code
  - Call type (voice/SMS)
  - LAC/CELL information
- **Distributed writing** to multiple directories
- **Automatic file rotation**

Offline traffic generates CDR files and transfers them to a directory for further processing

### Online Mode
#### Diameter Protocol Support (RFC 4006)
- **Credit-Control requests** (CCR)
- **AVP support** (Attribute-Value Pairs)
- **Automatic fallback to offline** on response codes 4011, 4522, 4012
- **Reconnection** on connection loss
- **Watchdog mechanism** (DWR/DWA)

When receiving code **4011** or **Continum** for Diameter, an offline CDR is generated

#### CAMEL Protocol Support (SCP via TLV decoder + ASN)
- **SCP server** for processing CAMEL messages
- **Supported message types**:
  - `TYPE_AUTHORIZESMS_REJECT/CONFIRM`
  - `TYPE_AUTHORIZEVOICE_REJECT/CONFIRM`
  - `TYPE_ENDSMS_RESP`
  - `TYPE_ENDVOICE_RESP`
- **TLV decoding** + ASN.1
- **Priority routing** CAMEL+BRT

When receiving code **TYPE_AUTHORIZESMS_REJECT** for CAMEL(SCP), an offline CDR is generated

## Architecture

### Components
1. **Load Generator** - creates test calls based on CSV subscriber pools
2. **Thread Manager** - manages worker goroutines for parallel processing
3. **Diameter Client** - interaction with BRT (Balance and Rating Tool)
4. **CAMEL Server** - online-CDR processing via SCP
5. **Monitoring** - statistics and metrics collection (InfluxDB support)
6. **Logging** - structured logging with rotation

### Key Features
- **Multithreading** - configurable number of worker goroutines
- **Load balancing** - automatic creation of additional threads when speed drops
- **Real-time statistics** - monitoring of speed and response codes
- **Data pool support** - CSV files with subscribers and LAC/CELL information
- **JSON configuration** - flexible task and parameter configuration

## Installation and Running

### Requirements
- Go 1.21 or higher
- Access to BRT servers (for online mode)
- CSV files with subscriber data

### Build
```bash
git clone https://github.com/egorkovalchuk/go-cdrgenerator.git
cd go-cdrgenerator
go build -o generator cmd/generator.go
```

### Configuration
Before running, configure `config.json`:
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

## Usage

### Command Line Parameters

#### Basic parameters:
```
-config string     Configuration file (default "config.json")
-debug             Debug mode
-d                 Start in daemon mode
-s                 Stop daemon
-v                 Print version
```

#### Operation modes:
```
-brt               Connect to BRT via Diameter protocol
-camel             Start SCP server for CAMEL protocol
-file              Write CDR to files (offline mode)
```

#### BRT parameters:
```
-brtlist value     List of tasks for BRT work (e.g., "local,roam")
                   Available values: local, roam, and others from config.json
```

#### Test parameters:
```
-rm                Delete all files in directories after work (test)
-slow              Uniform message sending with delays
-slow_camel        Send CAMEL messages once every 10 seconds
-thread            Allow starting additional threads
```

### Usage Examples

#### 1. Offline CDR file generation:
```bash
./generator -debug -file -config config.json
```

#### 2. Online mode with Diameter:
```bash
./generator -debug -d -brt -brtlist local,roam -config config.json
```

#### 3. Online mode with CAMEL:
```bash
./generator -debug -d -camel -config config.json
```

#### 4. Mixed mode (Diameter + CAMEL):
```bash
./generator -debug -d -camel -brt -brtlist local -config config.json
```

#### 5. Stop daemon:
```bash
./generator -s
```

#### 6. Test mode with cleanup:
```bash
./generator -debug -file -rm -config config.json
```

#### Warning
CAMEL works only in daemon mode. When starting CAMEL+BRT -> priority sending via CAMEL(SCP)

## Data Formats

### CSV Subscriber Pool (example):
```csv
MSISDN;IMSI;CallsCount
79161234567;250012345678901;10
79167654321;250098765432109;5
```

### CDR Record Format:
```
ID;Caller;Callee;StartTime;EndTime;Duration;CallType;Result;Termination;LAC;CELL
550e8400-e29b-41d4-a716-446655440000;79161234567;79001234567;2024-01-15T10:30:00Z;2024-01-15T10:30:45Z;45;01;0;normal;1001;101
```

### LAC/CELL Pool (example):
```csv
LAC;CELL
1001;101
1001;102
1002;201
```

## Monitoring and Statistics

### Real-time Metrics:
- **Generation speed** (operations/second)
- **Number of sent/received messages**
- **Diameter/CAMEL response codes**
- **Call type statistics**

### InfluxDB Integration:
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

## Operational Features

### Processing Priorities:
1. **CAMEL** (if CAMEL clients are connected)
2. **Diameter** (if BRT servers are connected and task is in brtlist)
3. **Files** (offline mode)

### Error Handling:
- **Diameter**: Codes 4011, 4522, 4012 → offline CDR generation
- **CAMEL**: `TYPE_AUTHORIZESMS_REJECT` → offline CDR generation
- **Unknown subscribers** (5030) → skip record

### Memory Management:
- **Subscriber pool** loaded into memory
- **Offline-CDR cache** for pending responses
- **Channels** for inter-goroutine communication

## Compatibility

### Supported Protocols:
- **Diameter** (RFC 4006, 3588)
- **CAMEL Phase 3+** (ETSI TS 129 078)

### Tested With:
- Telecom operator billing systems
- BRT (Balance and Rating Tool)
- SCP (Service Control Point) servers

## Security

- **PID files** for daemon management
- **Access rights check** for write directories
- **Contexts** for graceful shutdown
- **Logging** of all operations

## Development

### Project Structure:
```
go-cdrgenerator/
├── cmd/
│   └── generator.go          # Main executable file
├── pkg/
│   ├── data/                # Data structures and utilities
│   ├── diameter/            # Diameter client
│   ├── tlv/                 # CAMEL (TLV) processing
│   ├── logger/              # Logging
│   ├── influx/              # InfluxDB integration
│   └── pid/                 # PID file management
├── config.json.example      # Configuration example
└── README.md               # Documentation
```

### Dependencies:
- `github.com/fiorix/go-diameter/v4` - Diameter protocol
- `github.com/egorkovalchuk/go-cdrgenerator/pkg/*` - internal packages

## License

MIT License.

## Author

Egor Kovalchuk

## Version

v0.5.8
