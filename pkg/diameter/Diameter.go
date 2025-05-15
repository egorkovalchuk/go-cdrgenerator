package diameter

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/egorkovalchuk/go-cdrgenerator/pkg/data"

	"github.com/fiorix/go-diameter/v4/diam"
	"github.com/fiorix/go-diameter/v4/diam/avp"
	"github.com/fiorix/go-diameter/v4/diam/datatype"
	"github.com/fiorix/go-diameter/v4/diam/dict"
	"github.com/fiorix/go-diameter/v4/diam/sm"
)

type DiamCH struct {
	TaskName string
	Message  *diam.Message
}

// Инициализация клиента
func Client(mux *sm.StateMachine) *sm.Client {
	return &sm.Client{
		Dict:               Default, //dict.Default,
		Handler:            mux,
		MaxRetransmits:     3,
		RetransmitInterval: time.Second,
		EnableWatchdog:     false, // Реализован на стороне приложения
		WatchdogInterval:   5 * time.Second,

		AuthApplicationID: []*diam.AVP{
			//AVP Auth-Application-Id (код 258) имеет тип Unsigned32 и используется для публикации поддержки  Authentication and Authorization части diameter приложения (см. Section 2.4).
			//Если AVP Auth-Application-Id присутствует в сообщении, отличном от CER и CEA, значение этого AVP ДОЛЖНО соответствовать Application-Id, присутствующему в заголовке этого сообщения Diameter.
			diam.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(4)), // RFC 4006
		},
		AcctApplicationID: []*diam.AVP{
			// Acct-Application-Id AVP (AVP-код 259) имеет тип Unsigned32 и используется для публикации поддержки  Accountingand части diameter приложения (см. Section 2.4)
			//Если AVP Acct-Application-Id присутствует в сообщении, отличном от CER и CEA, значение этого AVP ДОЛЖНО соответствовать Application-Id, присутствующему в заголовке этого сообщения Diameter.
			diam.NewAVP(avp.AcctApplicationID, avp.Mbit, 0, datatype.Unsigned32(3)), //3
		},

		//Vendor-Specific-Application-Id AVP
		//Vendor-Specific-Application-Id AVP (код 260) имеет тип Grouped и используется для публикации поддержки vendor-specific Diameter-приложения. Точно один экземпляр Auth-Application-Id или Acct-Application-Id AVP ДОЛЖЕН присутствовать в составе этого AVP. Идентификатор приложения, переносимый либо Auth-Application-Id, либо Acct-Application-Id AVP, ДОЛЖЕН соответствовать идентификатору приложения конкретного поставщика, описанному в (Section 11.3 наверное 5.3). Он ДОЛЖЕН также соответствовать идентификатору приложения, присутствующему в заголовке Diameter сообщений, за исключением  сообщении CER или CEA.
		//
		//AVP Vendor-Id - это информационный AVP, относящийся к поставщику, который может иметь авторство конкретного приложения Diameter. Он НЕ ДОЛЖЕН использоваться в качестве средства определения отдельного пространства идентификаторов Application-Id.
		//
		//AVP Vendor-Specific-Application-Id  ДОЛЖЕН быть установлен как можно ближе к заголовку Diameter.
		//
		//     AVP Format
		//      <Vendor-Specific-Application-Id> ::= < AVP Header: 260 >
		//                                           { Vendor-Id }
		//                                           [ Auth-Application-Id ]
		//                                          [ Acct-Application-Id ]
		//AVP Vendor-Specific-Application-Id  ДОЛЖЕН содержать только один из идентификаторов Auth-Application-Id или Acct-Application-Id. Если AVP Vendor-Specific-Application-Id получен без одного из этих двух AVP, то получатель ДОЛЖЕН вернуть ответ с Result-Code DIAMETER_MISSING_AVP. В ответ СЛЕДУЕТ также включить Failed-AVP, который ДОЛЖЕН содержать пример AVP Auth-Application-Id и AVP Acct-Application-Id.
		//
		//Если получен AVP Vendor-Specific-Application-Id, содержащий оба идентификатора Auth-Application-Id и Acct-Application-Id, то получатель ДОЛЖЕН выдать ответ с Result-Code DIAMETER_AVP_OCCURS_TOO_MANY_TIMES. В ответ СЛЕДУЕТ также включить два Failed-AVP, которые содержат полученные AVP Auth-Application-Id и Acct-Application-Id.
		VendorSpecificApplicationID: []*diam.AVP{
			diam.NewAVP(avp.VendorSpecificApplicationID, avp.Mbit, 0, &diam.GroupedAVP{
				AVP: []*diam.AVP{
					diam.NewAVP(avp.VendorID, avp.Mbit, 0, datatype.Unsigned32(PETER_SERVICE_VENDOR_ID)),
					diam.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(4)),
					//diam.NewAVP(avp.AcctApplicationID, avp.Mbit, 0, datatype.Unsigned32(4)),
				},
			}),
		},
		SupportedVendorID: []*diam.AVP{
			diam.NewAVP(avp.VendorSpecificApplicationID, avp.Mbit, 0, &diam.GroupedAVP{
				AVP: []*diam.AVP{
					diam.NewAVP(avp.VendorID, avp.Mbit, 0, datatype.Unsigned32(PETER_SERVICE_VENDOR_ID)),
					diam.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(4)),
				},
			}),
		},
	}
}

// Формирование сообщения
func CreateCCREventMessage(Msisdn data.RecTypePool, date time.Time, RecordType data.RecTypeRatioType, dict *dict.Parser) (*diam.Message, string, error) {
	// Описание что добавить RatingGroup - может быть списком

	sid := "session;" + strconv.Itoa(int(rand.Uint32()))
	diam_message := diam.NewRequest(COMMAND_CODE_Credit_Control, 4, dict)
	diam_message.NewAVP(avp.SessionID, avp.Mbit, 0, datatype.UTF8String(sid))

	//{ Auth-Application-Id }
	diam_message.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(4))
	//{ Service-Context-Id } из конфига БРТ для каждого лоджик кола
	if RecordType.ServiceContextId != "" {
		diam_message.NewAVP(avp.ServiceContextID, avp.Mbit, 0, datatype.UTF8String(RecordType.ServiceContextId))
	} else {
		return nil, "", errors.New("not use empty ServiceContextId")
	}
	//{ CC-Request-Type }
	// Используется тип 4 (просто событие)
	// 1- инициация, 2 обновление 3 - завершение
	diam_message.NewAVP(avp.CCRequestType, avp.Mbit, 0, datatype.Enumerated(REQUEST_TYPE_EVENT_REQUEST))
	//{ CC-Request-Number } Растет от 0 до .. в зависимости от текущей сессии
	diam_message.NewAVP(avp.CCRequestNumber, avp.Mbit, 0, datatype.Unsigned32(0))

	//Передаем идентификатор и имси абонента
	diam_message.NewAVP(avp.SubscriptionID, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.SubscriptionIDType, avp.Mbit, 0, datatype.Enumerated(0)),
			diam.NewAVP(avp.SubscriptionIDData, avp.Mbit, 0, datatype.UTF8String(Msisdn.Msisdn)), //"79251470282")),
		},
	})
	diam_message.NewAVP(avp.SubscriptionID, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.SubscriptionIDType, avp.Mbit, 0, datatype.Enumerated(1)),
			diam.NewAVP(avp.SubscriptionIDData, avp.Mbit, 0, datatype.UTF8String(Msisdn.IMSI)), //"250020153589056")),
		},
	})
	diam_message.NewAVP(avp.UserName, avp.Mbit, 0, datatype.UTF8String(Msisdn.Msisdn)) //"79251470282"))

	//{ Event-Timestamp }  Время события
	diam_message.NewAVP(avp.EventTimestamp, avp.Mbit, 0, datatype.Time(time.Now()))
	//{ Multiple-Services-Indicator }
	diam_message.NewAVP(avp.MultipleServicesIndicator, avp.Mbit, 0, datatype.Enumerated(1))
	// { Requested-Action } Безусловное списание 0
	diam_message.NewAVP(avp.RequestedAction, avp.Mbit, 0, datatype.Enumerated(0))
	//diam.NewAVP(avp.RatingGroup, avp.Mbit, 0, datatype.Unsigned32(0))

	// { Multiple-Services-Credit-Control } Используется для сессий
	// SMS
	switch RecordType.MeasureType {
	case "SPECIFIC":
		{
			diam_message.NewAVP(avp.MultipleServicesCreditControl, avp.Mbit, 0, &diam.GroupedAVP{
				AVP: []*diam.AVP{
					// Requested-Service-Unit
					diam.NewAVP(avp.RequestedServiceUnit, avp.Mbit, 0, &diam.GroupedAVP{
						AVP: []*diam.AVP{
							//Для СМС.
							diam.NewAVP(avp.CCServiceSpecificUnits, avp.Mbit, 0, datatype.Unsigned64(1)),
						},
					}),
					//{ Service-Identifier }
					// diam_message.NewAVP(avp.ServiceIdentifier, avp.Mbit, 0, datatype.Unsigned32(60)),
					diam.NewAVP(avp.RatingGroup, avp.Mbit, 0, datatype.Unsigned32(RecordType.RatingGroup)),
				},
			})
			// { Service-Information }
			diam_message.NewAVP(avp.ServiceInformation, avp.Mbit, 10415, &diam.GroupedAVP{
				AVP: []*diam.AVP{
					diam.NewAVP(avp.SMSInformation, avp.Mbit, 10415, &diam.GroupedAVP{
						AVP: []*diam.AVP{
							diam.NewAVP(avp.SMSNode, avp.Mbit, 10415, datatype.Enumerated(0)),
						},
					}),
				},
			})
		}
	case "SECONDS":
		{
			//голос
			diam_message.NewAVP(avp.MultipleServicesCreditControl, avp.Mbit, 0, &diam.GroupedAVP{
				AVP: []*diam.AVP{
					// Requested-Service-Unit
					diam.NewAVP(avp.RequestedServiceUnit, avp.Mbit, 0, &diam.GroupedAVP{
						AVP: []*diam.AVP{
							//Для интернета октеты.
							diam.NewAVP(avp.CCTime, avp.Mbit, 0, datatype.Unsigned32(rand.Intn(999))),
						},
					}),
					diam.NewAVP(avp.RatingGroup, avp.Mbit, 0, datatype.Unsigned32(RecordType.RatingGroup)),
				},
			})
			// { Service-Information }
			diam_message.NewAVP(avp.ServiceInformation, avp.Mbit, 10415, &diam.GroupedAVP{
				AVP: []*diam.AVP{
					diam.NewAVP(avp.PSInformation, avp.Mbit, 10415, &diam.GroupedAVP{
						AVP: []*diam.AVP{
							diam.NewAVP(avp.CallingStationID, avp.Mbit, 0, datatype.UTF8String("internet.volume.pcef.vpcef")),
							//diam.NewAVP(avp. , avp.Mbit, 10415, datatype.IPv4{})
							//3GPP-PDP-Type
							//diam.NewAVP(avp.SGSNAddress, avp.Mbit, 10415, datatype.IPv4{}),
						},
					}),
				},
			})
		}
	default:
		{
			//Интернет
			diam_message.NewAVP(avp.MultipleServicesCreditControl, avp.Mbit, 0, &diam.GroupedAVP{
				AVP: []*diam.AVP{
					// Requested-Service-Unit
					diam.NewAVP(avp.RequestedServiceUnit, avp.Mbit, 0, &diam.GroupedAVP{
						AVP: []*diam.AVP{
							//Для интернета октеты.
							diam.NewAVP(avp.CCTotalOctets, avp.Mbit, 0, datatype.Unsigned64(rand.Intn(1000))),
						},
					}),
					diam.NewAVP(avp.RatingGroup, avp.Mbit, 0, datatype.Unsigned32(RecordType.RatingGroup)),
				},
			})
			// { Service-Information }
			diam_message.NewAVP(avp.ServiceInformation, avp.Mbit, 10415, &diam.GroupedAVP{
				AVP: []*diam.AVP{
					diam.NewAVP(avp.PSInformation, avp.Mbit, 10415, &diam.GroupedAVP{
						AVP: []*diam.AVP{
							diam.NewAVP(avp.CallingStationID, avp.Mbit, 0, datatype.UTF8String("internet.volume.pcef.vpcef")),
							//diam.NewAVP(avp. , avp.Mbit, 10415, datatype.IPv4{})
							//3GPP-PDP-Type
							//diam.NewAVP(avp.SGSNAddress, avp.Mbit, 10415, datatype.IPv4{}),
						},
					}),
				},
			})
		}
	}

	return diam_message, sid, nil
}

// Обработчик ответа, возвращает код ответа и сессию
func ResponseDiamHandler(message *diam.Message, f func(logtext interface{}), debug bool) (int, string) {

	var err error
	// универсальный формирование ответа
	/*op := ""
	cmd, err := message.Dictionary().FindCommand(
		message.Header.ApplicationID,
		message.Header.CommandCode,
	)
	if err != nil {
		op += "unknown_command"
	} else {
		op += cmd.Short + "A"
	}*/
	//ans := "DIAM: Answer " + op + " code:"

	// выделение кода ответа
	mm, err := message.FindAVPs(268, 0)
	if err != nil {
		f(message)
	}

	resp_code := 0
	s := 0
	for _, i := range mm {
		t := ConvertType(i)
		if s, err = strconv.Atoi(t); s > resp_code {
			if err == nil {
				resp_code = s
			}
		}
	}

	// Текст ошибки
	mm, _ = message.FindAVPs(avp.ErrorMessage, 0)
	for _, i := range mm {
		f(" ResponseDiamHandler: " + ConvertType(i))
	}

	// Получение SID
	if message.Header.CommandCode == 272 {
		m, r1 := message.FindAVP(263, 0)
		if r1 != nil {
			f(message)
		}
		if m.String() == "" {
			f(" ResponseDiamHandler: " + ConvertType(m))
		}
		return s, ConvertType(m)
	} else {
		return s, ""
	}
}

// Конвертер для ошибок в строку
func ConvertType(m *diam.AVP) string {
	switch m.Data.Type() {
	case 16:
		replacer := strings.NewReplacer("Unsigned32{", "", "}", "")
		return replacer.Replace(m.Data.String())
	case 15:
		re := regexp.MustCompile(`UTF8String{(.*)},.*`)
		return re.FindStringSubmatch(m.Data.String())[1]
	default:
		return m.Data.String()
	}

}

// переподключение
func Reconnect(cli *sm.Client, addr string, logFunc func(interface{})) diam.Conn {
	var retryCount int

	for {
		brt_connect, err := Dial(cli, addr, "", "", false, "tcp")
		logFunc("Reconnect diameter client " + addr + " retry " + fmt.Sprint(retryCount))
		if err != nil {
			retryCount++
			logFunc(fmt.Sprintf("Error connetct to %s: %v", addr, err))
			if retryCount > 5 {
				logFunc("Maximum number of connection attempts reached")
				return nil
			}
			time.Sleep(30 * time.Second)
			continue
		} else {
			logFunc(fmt.Sprintf("Successful connetct to %s", addr))
			return brt_connect
		}
	}
}

// Кусок для диаметра
// Определение шифрование соединения
func Dial(cli *sm.Client, addr, cert, key string, ssl bool, networkType string) (diam.Conn, error) {
	if ssl {
		return cli.DialNetworkTLS(networkType, addr, cert, key, nil)
	}
	return cli.DialNetwork(networkType, addr)
}
