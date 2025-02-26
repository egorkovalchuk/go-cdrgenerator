package tlv

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/egorkovalchuk/go-cdrgenerator/data"
)

const (
	Bool   = 1
	Uint8  = 1
	Uint16 = 2
	Uint32 = 4
	Uint64 = 8

	// Длина кадра
	TypeSize     = 2
	LengthSize   = 2
	SequenceSize = 4
	// параметры
	TagSize         = 2
	LengthSizeParam = 2
)

// Описание пареметров в пакете, Optional - является ли данный параметр обязательным
type camel_param_desc struct {
	Name     string
	Type     string
	MaxLen   int
	Tag      uint16
	Optional bool
}

// Структура входящего пакета
type Camel_tcp struct {
	LengthTCP uint16
	Type      uint16
	Sequence  uint32
	//сделать map
	Frame map[uint16]Camel_tcp_param
}

// Структура параметров в пакете
type Camel_tcp_param struct {
	Tag          uint16
	LengthParams uint16
	Param        []byte
	Type         string
}

// Описание типов и их длины в пакетах
type camel_type_len struct {
	Type   string
	MaxLen int
	Static bool
}

//| Length | Type | Sequence | Parameters |
//|   2Б   |  2Б  |    4Б    |            |
//    Length: длина всего сообщения.
//    Type: тип сообщения.
//    Sequence: id запроса. В ответе должен быть указан он же.

//Parameters:
//| Tag | Length | Value |
//|  2Б |   2Б   |       |
//    Tag: код параметра.
//    Length: длина значения.
//	Value: значение параметра.

// Инициализация пакета
func NewCamelTCP() Camel_tcp {
	return Camel_tcp{
		Frame: make(map[uint16]Camel_tcp_param),
	}
}

// Удаление/деструктор
func (p *Camel_tcp) DeleteCamelTCP() {}

func NewCamelTCPParam() Camel_tcp_param {
	return Camel_tcp_param{}
}

// Декодирвоание пакета
func (p *Camel_tcp) Decoder(r []byte) error {
	defer func() {
		if c := recover(); c != nil {
			LogChannel <- LogStruct{"INFO: Decoder", "Error TLV parsing"}
			LogChannel <- LogStruct{"INFO: Decoder", r}
		}
	}()

	p.LengthTCP = binary.BigEndian.Uint16(r[0:2])
	p.Type = binary.BigEndian.Uint16(r[2:4])
	p.Sequence = binary.BigEndian.Uint32(r[4:8])

	if 8 >= int(p.LengthTCP) {
		return nil
	}

	err := p.DecoderChunk(r, 8)
	if err != nil {
		return err
	}
	return nil
}

// Декодирвоание пакета буффера
func (p *Camel_tcp) DecoderBuffer(r []byte) (t []byte, cont int, err error) {
	defer func() {
		if c := recover(); c != nil {
			LogChannel <- LogStruct{"INFO: DecoderBuffer", "Error TLV parsing"}
			LogChannel <- LogStruct{"INFO: DecoderBuffer", r}
			LogChannel <- LogStruct{"INFO: DecoderBuffer", c}
		}
	}()

	rl := len(r)

	if rl < 8 {
		return r, -1, nil
	}
	cont = 0
	p.LengthTCP = binary.BigEndian.Uint16(r[0:2])
	p.Type = binary.BigEndian.Uint16(r[2:4])
	p.Sequence = binary.BigEndian.Uint32(r[4:8])

	i := int(p.LengthTCP)

	// cont определяет размер соответствия буфера и пакета
	// 1 больше декодируем камел пакет и возворащаем
	// 0 равно просто декодируем
	// -1 меньше - просто возвращаем остаток
	switch {
	case rl > i:
		t = r[i:]
		cont = 1
		if i > 8 {
			err = p.DecoderChunk(r[0:i], 8)
		}
		if err != nil {
			return nil, cont, err
		}
		return t, cont, nil
	case rl < i:
		cont = -1
		return r, -1, nil
	case rl == i && i == 8:
		return nil, cont, nil
	case rl == i && i != 8:
		err = p.DecoderChunk(r[0:i], 8)
		if err != nil {
			return nil, cont, err
		}
		return nil, cont, nil
	}
	return
}

// идем по всем параметрам пришедшего пакета для декодирования
func (p *Camel_tcp) DecoderChunk(r []byte, n int) error {

	var err error
	tmp := Camel_tcp_param{}
	tmp.Tag = binary.BigEndian.Uint16(r[n : n+2])
	tmp.LengthParams = binary.BigEndian.Uint16(r[n+2 : n+4])
	nn := n + 4 + int(tmp.LengthParams)
	tmp.Param = r[n+4 : nn]
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	if nn < int(p.LengthTCP) {
		err = p.DecoderChunk(r, nn)
	} else {
		return nil
	}
	if err != nil {
		LogChannel <- LogStruct{"ERROR", err}
		return err
	}
	return nil
}

// Возврат значения параметра строкой
func (d *Camel_tcp_param) ParamString() string {
	if d.Type == "string" {
		return string(d.Param)
	} else if d.Type == "integer" {
		s := binary.BigEndian.Uint32(d.Param)
		return fmt.Sprint(s)
	} else if d.Type == "byte" {
		return fmt.Sprint(d.Param)
	}
	return ""
}

// как реализовано в пакете go
/*func (bigEndian) AppendUint32(b []byte, v uint32) []byte {
	return append(b,
		byte(v>>24),
		byte(v>>16),
		byte(v>>8),
		byte(v),
	)
}*/

// Кодирование сообщений
func (p *Camel_tcp) Encoder() ([]byte, error) {
	var err error
	p.LengthTCP, err = p.LenghtTCP()

	if err != nil {
		LogChannel <- LogStruct{"INFO: ENCODER1", p}
	}

	var tmp []byte

	tmp = binary.BigEndian.AppendUint16(tmp, p.LengthTCP)
	tmp = binary.BigEndian.AppendUint16(tmp, p.Type) //AppendUint16(tmp, p.Type)
	tmp = binary.BigEndian.AppendUint32(tmp, p.Sequence)

	for _, i := range p.Frame {
		switch i.Type {
		case "integer":
			tmp = binary.BigEndian.AppendUint16(tmp, i.Tag)
			tmp = binary.BigEndian.AppendUint16(tmp, uint16(camel_type_map[i.Type].MaxLen))
			tmp = append(tmp, i.Param...)
		case "string":
			tmp = binary.BigEndian.AppendUint16(tmp, i.Tag)
			tmp = binary.BigEndian.AppendUint16(tmp, uint16(i.LengthParams))
			tmp = append(tmp, i.Param...)
		case "byte":
			tmp = binary.BigEndian.AppendUint16(tmp, i.Tag)
			tmp = binary.BigEndian.AppendUint16(tmp, uint16(camel_type_map[i.Type].MaxLen))
			tmp = append(tmp, i.Param...)
		case "buffer":
			tmp = binary.BigEndian.AppendUint16(tmp, i.Tag)
			tmp = binary.BigEndian.AppendUint16(tmp, uint16(i.LengthParams))
			tmp = append(tmp, i.Param...)
		case "short":
			tmp = binary.BigEndian.AppendUint16(tmp, i.Tag)
			tmp = binary.BigEndian.AppendUint16(tmp, uint16(camel_type_map[i.Type].MaxLen))
			tmp = append(tmp, i.Param...)
		case "boolean":
			tmp = binary.BigEndian.AppendUint16(tmp, i.Tag)
			tmp = binary.BigEndian.AppendUint16(tmp, uint16(camel_type_map[i.Type].MaxLen))
			tmp = append(tmp, i.Param...)
		}
	}

	if err != nil {
		LogChannel <- LogStruct{"ERROR", err}
	}

	return tmp, nil
}

func (p *Camel_tcp) LenghtTCP() (uint16, error) {
	var tmp int
	tmp = 8
	for _, i := range p.Frame {
		tmp += 4
		if !camel_type_map[i.Type].Static {
			tmp += len(i.Param)
		} else {
			tmp += camel_type_map[i.Type].MaxLen
		}
	}
	return uint16(tmp), nil
}

func (p *Camel_tcp) AuthorizeSMS_req(msisdn string, imsi string, ServiceCode string, msisdnB string, lc data.RecTypeLACPool, s *Server) ([]byte, error) {
	var err error
	p.Sequence = s.Sec + uint32(1)
	p.Type = TYPE_AUTHORIZESMS_REQ

	//SessionID string О Идентификатор сессии
	tmp := NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x002C].Tag
	tmp.Param = NewCamelSessionID(msisdn, byte(0), s)
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	// ServiceKey integer О Идентификатор запрошенной услуги
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x002B].Tag
	// Поменять на из параметров
	tmp.Param = binary.BigEndian.AppendUint32([]byte{}, 1)
	tmp.LengthParams = uint16(camel_params_map[0x002B].MaxLen)
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//CAPProtocolVersion byte O Версия протокола CAP (вызывающей стороны). 2 – CAMEL2 3 – CAMEL3
	//Определяется SCP для каждого вызова из диалоговой части TCAP (параметр ApplicationContentName).
	//Необходим для определения возможности в BRT поддержки дополнительных особенностей CAMEL фазы 3, например, FCI
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0042].Tag
	tmp.Param = []byte{3}
	tmp.LengthParams = uint16(camel_params_map[0x0042].MaxLen)
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	// EventTypeSMS byte O Идентификатор события активировавшего SMS DP
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0015].Tag
	tmp.Param = []byte{1}
	tmp.LengthParams = uint16(camel_params_map[0x0015].MaxLen)
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//SMSCAddressNumber
	if s.cfg.Camel_SMSAddress != "" {
		tmp = NewCamelTCPParam()
		tmp.Tag = camel_params_map[0x002E].Tag
		tmp.Param = []byte(s.cfg.Camel_SMSAddress)
		tmp.LengthParams = uint16(len(tmp.Param))
		tmp.Type = camel_params_map[tmp.Tag].Type
		p.Frame[tmp.Tag] = tmp

		//SMSCAddressInformation buffer О Параметры SMSCAddress
		tmp = NewCamelTCPParam()
		tmp.Tag = camel_params_map[0x002D].Tag
		tmp.Param = []byte{1, 1}
		tmp.LengthParams = uint16(len(tmp.Param))
		tmp.Type = camel_params_map[tmp.Tag].Type
		p.Frame[tmp.Tag] = tmp
	}

	//LocationInformationMSC
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0019].Tag
	tmp.Param = s.LocationMSC(lc)
	//tmp.Param = []byte{0xbf, 0x34, 0x0b, 0xa3, 0x09, 0x80, 0x07, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00}
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//TimeAndTimezone
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0030].Tag
	tmp.Param = Stringtobytereverse(time.Now().Format("20060102030405") + "00")
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//IMSI
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0017].Tag
	tmp.Param = []byte(imsi)
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//MSISDNNumber
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0020].Tag
	tmp.Param = []byte(msisdn)
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//CallingPartyNumber
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x000A].Tag
	tmp.Param = []byte(msisdn)
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//CallingPartyNumberInformation
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x000B].Tag
	tmp.Param = []byte{1, 1, 0}
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//DestinationRoutingNumber
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0010].Tag
	tmp.Param = []byte(msisdnB)
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//DestinationRoutingNumberInformation
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0011].Tag
	tmp.Param = []byte{1, 1}
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	p.LengthTCP, err = p.LenghtTCP()

	// Возвращаем id сессии
	return p.Frame[0x002C].Param[0:12], err
}

// По умолчанию генератор отправляет что все СМС доставлены
func (p *Camel_tcp) EndSMS_req(sid []byte, s *Server) error {
	var err error
	p.Sequence = s.Sec + uint32(1)
	p.Type = TYPE_ENDSMS_REQ

	//SessionID string О Идентификатор сессии
	tmp := NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x002C].Tag
	tmp.Param = sid
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	// EndReason byte О Причина завершения транзакции
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x003A].Tag
	tmp.Param = []byte{0x00}
	tmp.LengthParams = uint16(camel_params_map[0x002B].MaxLen)
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	// SMSStatus byte Опц Результат посылки SMS
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x002F].Tag
	tmp.Param = []byte{0x00}
	tmp.LengthParams = uint16(camel_params_map[0x002B].MaxLen)
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	return err
}

// Авторизация Звонка
func (p *Camel_tcp) AuthorizeVoice_req(msisdn string, imsi string, ServiceCode string, msisdnB string, lc data.RecTypeLACPool, s *Server) ([]byte, error) {
	var err error

	p.Sequence = s.Sec + uint32(1)
	p.Type = TYPE_AUTHORIZEVOICE_REQ

	//SessionID string О Идентификатор сессии
	tmp := NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x002C].Tag
	tmp.Param = NewCamelSessionID(msisdn, byte(0), s)
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	// ServiceKey integer О Идентификатор запрошенной услуги
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x002B].Tag
	// Поменять на из параметров
	tmp.Param = binary.BigEndian.AppendUint32([]byte{}, 1)
	tmp.LengthParams = uint16(camel_params_map[0x002B].MaxLen)
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//CAPProtocolVersion byte O Версия протокола CAP (вызывающей стороны). 2 – CAMEL2 3 – CAMEL3
	//Определяется SCP для каждого вызова из диалоговой части TCAP (параметр ApplicationContentName).
	//Необходим для определения возможности в BRT поддержки дополнительных особенностей CAMEL фазы 3, например, FCI
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0042].Tag
	tmp.Param = []byte{3}
	tmp.LengthParams = uint16(camel_params_map[0x0042].MaxLen)
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//IMSI
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0017].Tag
	tmp.Param = []byte(imsi)
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//MSISDNNumber
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0020].Tag
	tmp.Param = []byte(msisdn)
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//ServiceCode Integer О Код базовой
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x002A].Tag
	// меняем в зависимости от параметра - МО (01) или MT (02)
	n, _ := strconv.Atoi(ServiceCode)
	tmp.Param = binary.BigEndian.AppendUint32([]byte{}, uint32(n))
	tmp.LengthParams = uint16(camel_params_map[0x002A].MaxLen)
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//ServiceCodeType Byte О Тип кода базовой услуги
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x003F].Tag
	tmp.Param = []byte{1}
	tmp.LengthParams = uint16(camel_params_map[0x003F].MaxLen)
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//CallingPartyNumber
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x000A].Tag
	tmp.Param = []byte(msisdn)
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//CallingPartyNumberInformation
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x000B].Tag
	tmp.Param = []byte{1, 1, 0}
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	// Кому звоним
	// CalledPartyNumber String Опц Цифры номера вызываемого абонента
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0008].Tag
	tmp.Param = []byte(msisdnB)
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	// Кому звоним
	// CalledPartyNumberInformation Buffer Опц Параметры номера вызываемого
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0009].Tag
	tmp.Param = []byte{1, 1, 0}
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//TimeAndTimezone
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0030].Tag
	tmp.Param = Stringtobytereverse(time.Now().Format("20060102030405") + "00")
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	// LocationNumber String Опц Информация оместоположении(цифры номера)
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x001A].Tag
	tmp.Param = []byte("")
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	// EventTypeBCSM Byte О Идентификатор события активировавшего BCSM DP
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0014].Tag
	tmp.Param = []byte{2}
	tmp.LengthParams = uint16(camel_params_map[0x0014].MaxLen)
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	// LocationInformation buffer О Этот параметр содержит информацию о местоположении мобильного пользователя и возраст этой информации.
	// Данные должны быть представлены в ASN.1 формате (Sequence представление из InitialDP) без распаковки со стороны SCP
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0018].Tag
	tmp.Param = s.LocationMSC(lc)
	//tmp.Param = []byte{0xbf, 0x34, 0x0b, 0xa3, 0x09, 0x80, 0x07, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00}
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	tt := ""
	// MscAddressNumber String О Содержит параметр mscId присваиваемый GMSC/MSC
	if tt != "" {
		tmp = NewCamelTCPParam()
		tmp.Tag = camel_params_map[0x001E].Tag
		tmp.Param = []byte("")
		tmp.LengthParams = uint16(len(tmp.Param))
		tmp.Type = camel_params_map[tmp.Tag].Type
		p.Frame[tmp.Tag] = tmp

		// MscAddressInformation Buffer О Параметры MscAddress
		tmp = NewCamelTCPParam()
		tmp.Tag = camel_params_map[0x001D].Tag
		tmp.Param = []byte{1, 1, 0}
		tmp.LengthParams = uint16(len(tmp.Param))
		tmp.Type = camel_params_map[tmp.Tag].Type
		p.Frame[tmp.Tag] = tmp
	}

	// Возвращаем id сессии
	return p.Frame[0x002C].Param[0:12], err
}

// По умолчанию генератор отправляет что голосовой вызов завершен
func (p *Camel_tcp) EndVoice_req(sid []byte, s *Server) error {
	var err error
	p.Sequence = s.Sec + uint32(1)
	p.Type = TYPE_ENDVOICE_REQ

	//SessionID string О Идентификатор сессии
	tmp := NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x002C].Tag
	tmp.Param = sid
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	// EndReason byte О Причина завершения транзакции
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x003A].Tag
	tmp.Param = []byte{0x00}
	tmp.LengthParams = uint16(camel_params_map[0x002B].MaxLen)
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	// ReleaseCause integer Опц Код возврата со стороны MSC.
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0026].Tag
	tmp.Param = binary.BigEndian.AppendUint32([]byte{}, 0)
	tmp.LengthParams = uint16(camel_params_map[0x0026].MaxLen)
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	// CallAttemptElapsedTime integer О Интервал времени между окончанием процедур инициализации вызова (Connect или Continue)
	// и получением ответа вызывающей стороной отвызываемо
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0004].Tag
	tmp.Param = binary.BigEndian.AppendUint32([]byte{}, 0)
	tmp.LengthParams = uint16(camel_params_map[0x0004].MaxLen)
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//CallConnectedElapsedTime integer О Интервал времени между получением ответа от вызываемой стороны и освобожден
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x0005].Tag
	tmp.Param = binary.BigEndian.AppendUint32([]byte{}, 1360)
	tmp.LengthParams = uint16(camel_params_map[0x0005].MaxLen)
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp

	//CallStopTime String О Временная отметка момента освобождения вызова
	tmp = NewCamelTCPParam()
	tmp.Tag = camel_params_map[0x000E].Tag
	tmp.Param = []byte(time.Now().Format("01.02.2006 03:04:05"))
	tmp.LengthParams = uint16(len(tmp.Param))
	tmp.Type = camel_params_map[tmp.Tag].Type
	p.Frame[tmp.Tag] = tmp
	return err
}

// Получение нового значения сессии
func NewCamelSessionID(MSCGT string, id byte, s *Server) []byte {
	var tmp []byte
	var err error
	tmp = make([]byte, 16)
	ln, parity := divmod(len(MSCGT), 2)
	//var mscgt_rune string
	var bit_right, bit_left int
	for i := 0; i < 8; i++ {
		if i < ln {
			bit_left, err = strconv.Atoi(string(MSCGT[i*2]))
			if err != nil {
				LogChannel <- LogStruct{"ERROR: Stringtobytereverse", err}
			}
			bit_right, err = strconv.Atoi(string(MSCGT[i*2+1]))
			if err != nil {
				LogChannel <- LogStruct{"ERROR: NewSessionId", err}
			}
		} else {
			if parity != 0 && i == ln {
				bit_left, err = strconv.Atoi(string(MSCGT[i*2]))
				if err != nil {
					LogChannel <- LogStruct{"ERROR: NewSessionId", err}
				}
			} else {
				bit_left = 15
			}
			bit_right = 15
		}

		bit_left = bit_left << 4
		tt := bit_left + bit_right
		tmp[i] = byte(tt)
	}
	// Рандом (прилетает с SCP, для BRT и генератора фактически рандомное значение)
	// генерируем 4 байтное число и побитовым смещением записываем
	t := rand.Uint32()
	tmp[11] = byte(t >> 24)
	tmp[10] = byte(t >> 16)
	tmp[9] = byte(t >> 8)
	tmp[8] = byte(t)

	// SCP id
	tmp[12] = s.cfg.Camel_SCP_id
	//ид BRT
	tmp[13] = id
	return tmp
}

// Преобазование строки  байты - перевертыши
// 2006 - > 0x02 0x60
func Stringtobytereverse(t string) []byte {
	var tmp []byte
	var err error
	ln, parity := divmod(len(t), 2)
	tmp = make([]byte, ln+parity)

	var bit_right, bit_left int
	for i := 0; i <= ln; i++ {
		if i < ln {
			bit_left, err = strconv.Atoi(string(t[i*2]))
			if err != nil {
				LogChannel <- LogStruct{"ERROR: Stringtobytereverse", err}
			}
			bit_right, err = strconv.Atoi(string(t[i*2+1]))
			if err != nil {
				LogChannel <- LogStruct{"ERROR: Stringtobytereverse", err}
			}
			bit_right = bit_right << 4
			tt := bit_left + bit_right
			tmp[i] = byte(tt)
		} else {
			if parity != 0 && i == ln {
				bit_left, err = strconv.Atoi(string(t[i*2]))
				if err != nil {
					LogChannel <- LogStruct{"ERROR: Stringtobytereverse", err}
				}
				bit_right = 15
				bit_right = bit_right << 4
				tt := bit_left + bit_right
				tmp[i] = byte(tt)
			} else {
				bit_left = 15
			}
			bit_right = 15
		}
	}
	return tmp
}

// Преобазование строки  байты
// 2006 - > 0x20 0x06
func Stringtobyte(t string) []byte {
	var tmp []byte
	var err error
	ln, parity := divmod(len(t), 2)
	tmp = make([]byte, ln+parity)
	var bit_right, bit_left int
	for i := 0; i < ln; i++ {
		if i < ln {
			bit_left, err = strconv.Atoi(string(t[i*2]))
			if err != nil {
				LogChannel <- LogStruct{"ERROR: Stringtobytereverse", err}
			}
			bit_right, err = strconv.Atoi(string(t[i*2+1]))
			if err != nil {
				LogChannel <- LogStruct{"ERROR: Stringtobyte", err}
			}
		} else {
			if parity != 0 && i == ln {
				bit_left, err = strconv.Atoi(string(t[i*2]))
				if err != nil {
					LogChannel <- LogStruct{"ERROR: Stringtobyte", err}
				}
			} else {
				bit_left = 15
			}
			bit_right = 15
		}

		bit_left = bit_left << 4
		tt := bit_left + bit_right
		tmp[i] = byte(tt)
	}
	return tmp
}

// Целое и остаток от деления
func divmod(numerator, denominator int) (quotient, remainder int) {
	quotient = numerator / denominator
	remainder = numerator % denominator
	return
}

// Location MSC
func (s *Server) LocationMSC(lc data.RecTypeLACPool) []byte {

	// Генерация Location MSC
	buffer := s.LocationMSCbase

	tmp := binary.BigEndian.AppendUint16([]byte{}, uint16(lc.LAC))
	buffer = append(buffer, tmp...)
	tmp = binary.BigEndian.AppendUint16([]byte{}, uint16(lc.CELL))
	buffer = append(buffer, tmp...)

	buffer[1] = byte(len(buffer) - 2)

	return buffer
}

func (s *Server) InitMSC() {

	// Инициализация
	// Определяем не меняющаяся часть LocationMSC
	s.LocationMSCbase = append(s.LocationMSCbase, 0xa5) //Context, Constructed, 0x05
	s.LocationMSCbase = append(s.LocationMSCbase, 0x00) //Длина, заменим в конце

	// Дочерние
	// xVLR
	s.LocationMSCbase = append(s.LocationMSCbase, 0x81) //Context, Primitive, 0x01
	s.LocationMSCbase = append(s.LocationMSCbase, 0x07) //Длина
	//VLR Number
	//1... .... = extension: noExtersion (0x01)
	//.001 .... = natureOfAddressIndicator: International (0x01)
	//.... 0001 = numberingPlanIndicator: ISDN(Telephony)NumberingPlan (0x01)
	//ISDNString:79 28 99 00 09 1
	tmp := Stringtobytereverse(s.cfg.XVLR)
	s.LocationMSCbase = append(s.LocationMSCbase, 0x91) //Тип xVLR и его идентификатор
	s.LocationMSCbase = append(s.LocationMSCbase, tmp...)

	//CellIDorLAI
	s.LocationMSCbase = append(s.LocationMSCbase, 0xa3) //Context, Constructed, 0x03
	s.LocationMSCbase = append(s.LocationMSCbase, 0x09) //Длина
	s.LocationMSCbase = append(s.LocationMSCbase, 0x80) //Context, Primitive, 0x001
	s.LocationMSCbase = append(s.LocationMSCbase, 0x07) //Длина

	tmp = Stringtobytereverse(s.cfg.ContryCode)
	s.LocationMSCbase = append(s.LocationMSCbase, tmp...)
	tmp = Stringtobytereverse(s.cfg.OperatorCode)
	s.LocationMSCbase = append(s.LocationMSCbase, tmp...)
}
