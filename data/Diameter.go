package data

import (
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/fiorix/go-diameter/v4/diam"
	"github.com/fiorix/go-diameter/v4/diam/avp"
	"github.com/fiorix/go-diameter/v4/diam/datatype"
	"github.com/fiorix/go-diameter/v4/diam/dict"
)

// VENDOR_ID "Peter-Service Ltd."
const PETER_SERVICE_VENDOR_ID = 11971

const MAX_PACKET_SIZE = 4096

const delta_1970_1900 = 2208988800

const CHARGING_SCENARIO_ECUR = 0
const CHARGING_SCENARIO_SCUR = 1
const CHARGING_SCENARIO_FBC = 2

// MSG_ID
const MSG_ID_DWR = 0x80000118
const MSG_ID_DWA = 0x00000118
const MSG_ID_CER = 0x80000101
const MSG_ID_CEA = 0x00000101
const MSG_ID_ACR = 0x8000010F
const MSG_ID_ACA = 0x0000010F
const MSG_ID_CCR = 0x80000110
const MSG_ID_CCA = 0x00000110
const MSG_ID_DPR = 0x8000011A
const MSG_ID_DPA = 0x0000011A

//  COMMAND_CODE
const COMMAND_CODE_Capability_Exchange uint32 = 0x00000101
const COMMAND_CODE_Accounting_Control uint32 = 0x0000010F
const COMMAND_CODE_Credit_Control uint32 = 0x00000110
const COMMAND_CODE_Device_Watchdog uint32 = 0x00000118
const COMMAND_CODE_Disconnect_Peer uint32 = 0x0000011A

// AVP
const AVP_User_Name = 0x00000001                        //1
const AVP_GPP3_Charging_id = 0x00000002                 //2
const AVP_GPP3_PDP_Type = 0x00000003                    //3
const AVP_GPP3_GPRS_NEGOTIATED_QOS_PROFILE = 0x00000005 //5
const AVP_GPP3_IMSI_MCC_MNC = 0x00000008                //8
const AVP_GPP3_NSAPI = 0x0000000A                       //10
const AVP_GPP3_Selection_mode = 0x0000000C              //12
const AVP_GPP3_Charging_Characteristics = 0x0000000D    //13
const AVP_GPP3_SGSN_MCC_MNC = 0x00000012                //18
const AVP_User_Location_Info = 0x00000016               //22
const AVP_MS_TimeZone = 0x00000017                      //23
const AVP_Called_Station_Id = 0x0000001E                //30
const AVP_Event_Timestamp = 0x00000037                  //55
const AVP_Host_IP_Address = 0x00000101                  //257
const AVP_Auth_Application_Id = 0x00000102              //258
const AVP_Acct_Application_Id = 0x00000103              //259
const AVP_Rullbase_id = 0x00000106                      //262
const AVP_Session_id = 0x00000107                       //263
const AVP_Origin_Host = 0x00000108                      //264
const AVP_Vendor_Id = 0x0000010A                        //266
const AVP_Firmware_Revision = 0x0000010B                //267
const AVP_Result_Code = 0x0000010C                      //268
const AVP_Product_Name = 0x0000010D                     //269
const AVP_Origin_State_Id = 0x00000116                  //278
const AVP_Error_Message = 0x00000119                    //281
const AVP_Destination_Realm = 0x0000011B                //283
const AVP_RullSpaceSuggestion = 0x00000122              //290
const AVP_Destination_Host = 0x00000125                 //293
const AVP_Termination_Cause = 0x00000127                //295
const AVP_Origin_Realm = 0x00000128                     //296
const AVP_Inband_Security_Id = 0x0000012B               //299
const AVP_CC_Input_Octets = 0x0000019C                  //412
const AVP_CC_Money = 0x0000019D                         //413
const AVP_CC_Output_Octets = 0x0000019E                 //414
const AVP_CC_Request_Number = 0x0000019F                //415
const AVP_CC_Request_Type = 0x000001A0                  //416
const AVP_CC_Units = 0x000001A1                         //417
const AVP_CC_Time = 0x000001A4                          //420
const AVP_CC_Total_Octets = 0x000001A5                  //421
const AVP_Cost_Ubit = 0x000001A8                        //424
const AVP_Currency_Code = 0x000001A9                    //425
const AVP_Exponent = 0x000001AD                         //429
const AVP_Granted_Service_Unit = 0x000001AF             //431
const AVP_Rating_Group = 0x000001B0                     //432
const AVP_Requested_Action = 0x000001B4                 //436
const AVP_Requested_Service_Unit = 0x000001B5           //437
const AVP_Service_Identifier = 0x000001B7               //439
const AVP_Service_Parameter_Info = 0x000001B8           //440
const AVP_Service_Parameter_Type = 0x000001B9           // 441
const AVP_Service_Parameter_Value = 0x000001BA          // 442
const AVP_Subscription_Id = 0x000001BB                  //443
const AVP_Subscription_Id_Data = 0x000001BC             //444
const AVP_Unit_Value = 0x000001BD                       //445
const AVP_Used_Service_Unit = 0x000001BE                //446
const AVP_Value_Digits = 0x000001BF                     //447
const AVP_Validity_Time = 0x000001C0                    //448
const AVP_Subscription_Id_Type = 0x000001C2             //450
const AVP_Multiple_Service_Indication = 0x000001C7      //455
const AVP_Multiple_Services_Credit_Control = 0x000001C8 //456
const AVP_User_Equipment_Info = 0x000001CA              //458
const AVP_User_Equipment_Info_Type = 0x000001CB         //459
const AVP_User_Equipment_Info_Value = 0x000001CC        //460
const AVP_Service_Context_Id = 0x000001CD               //461
const AVP_Accounting_Record_Type = 0x000001E0           //480
const AVP_Accounting_Record_Number = 0x000001E5         //485
const AVP_GGSN_Address = 0x0000034F                     //847
const AVP_Service_Specific_Data = 0x0000035F            //863
const AVP_Reporting_Reason = 0x00000368                 //872
const AVP_Service_Information = 0x00000369              //873
const AVP_PS_Information = 0x0000036A                   //874
const AVP_MMS_Information = 0x0000036D                  //877

const AVP_Address_Data = 0x00000381   //897
const AVP_Address_Domain = 0x00000382 //898
const AVP_Address_Type = 0x00000383   //899

const AVP_User_Data = 0x00000385                 //901
const AVP_Bis_Subscriber_Id_Support = 0x00000388 //904

const AVP_Service_Voice_Information = 0x0000038A //906
const AVP_ATime_Zone = 0x0000038B                //907
const AVP_AMCC = 0x0000038C                      //908
const AVP_AMNC = 0x0000038D                      //909
const AVP_AArea = 0x0000038E                     //910
const AVP_ACell = 0x0000038F                     //911
const AVP_ARoaming = 0x00000390                  //912
const AVP_AGlobal_Title = 0x00000391             //913
const AVP_SubscriptionB = 0x00000392             //914
const AVP_BTime_Zone = 0x00000393                //915
const AVP_SubscriptionC = 0x00000395             //917 (Начиная с BRT_SRV_052.00)

const AVP_Account_Information = 0x00000396  //918
const AVP_Account_Type = 0x00000397         //919
const AVP_Service_Content_Type = 0x00000398 //920

const AVP_Charging_Rule_Base_Name = 0x000003EC //1004
const AVP_Domain_Name = 0x000004B0             //1200
const AVP_Recipient_Address = 0x000004B1       //1201
const AVP_Addressee_Type = 0x000004B8          //1208
const AVP_PDP_Address = 0x000004CB             //1227
const AVP_SGSN_Address = 0x000004CC            //1228

const AVP_Service_Specific_Info = 0x000004E1 //1249
const AVP_Service_Specific_Type = 0x000004E9 //1257

const AVP_SMS_Information = 0x000007D0         //2000
const AVP_Data_Coding_Scheme = 0x000007D1      //2001
const AVP_Destination_Interface = 0x000007D2   //2002
const AVP_Interface_Id = 0x000007D3            //2003
const AVP_Interface_Port = 0x000007D4          //2004
const AVP_Interface_Text = 0x000007D5          //2005
const AVP_Interface_Type = 0x000007D6          //2006
const AVP_SM_Message_Type = 0x000007D7         //2007
const AVP_Originator_SCCP_Address = 0x000007D8 //2008
const AVP_Originator_Interface = 0x000007D9    //2009
const AVP_Recipient_SCCP_Address = 0x000007DA  //2010
const AVP_Reply_Path_Requested = 0x000007DB    //2011
const AVP_SM_Discharge_Time = 0x000007DC       //2012
const AVP_SM_Protocol_Id = 0x000007DD          //2013
const AVP_SM_Status = 0x000007DE               //2014
const AVP_SM_User_Data_Header = 0x000007DF     //2015
const AVP_SMS_Node = 0x000007E0                //2016
const AVP_SMSC_Address = 0x000007E1            //2017
const AVP_Client_Address = 0x000007E2          //2018
const AVP_Number_Of_Messages_Sent = 0x000007E3 //2019

const AVP_Recipient_Info = 0x000007EA              //2026
const AVP_Originator_Received_Address = 0x000007EB //2027
const AVP_Recipient_Received_Address = 0x000007EC  //2028
const AVP_SM_Service_Type = 0x000007ED             //2029

// END_USER
const END_USER_END_USER_E164 = 0
const END_USER_END_USER_IMSI = 1
const END_USER_END_USER_SIP_URI = 2
const END_USER_END_USER_NAI = 3
const END_USER_END_USER_PRIVATE = 4

// ACCOUNTING_RECORD_TYPE
const ACCOUNTING_RECORD_TYPE_EVENT_RECORD = 1
const ACCOUNTING_RECORD_TYPE_START_RECORD = 2
const ACCOUNTING_RECORD_TYPE_INTERIM_RECORD = 3
const ACCOUNTING_RECORD_TYPE_STOP_RECORD = 4

// REQUEST_TYPE
const REQUEST_TYPE_UNKNOWN = 0
const REQUEST_TYPE_INITIAL_REQUEST = 1
const REQUEST_TYPE_UPDATE_REQUEST = 2
const REQUEST_TYPE_TERMINATION_REQUEST = 3
const REQUEST_TYPE_EVENT_REQUEST = 4

// REQUESTED_ACTION
const REQUESTED_ACTION_DIRECT_DEBITING = 0
const REQUESTED_ACTION_REFUND_ACCOUNT = 1
const REQUESTED_ACTION_CHECK_BALANCE = 2
const REQUESTED_ACTION_PRICE_ENQUIRY = 3

// SECURITY
const SECURITY_NO_INBAND_SECURITY = 0
const SECURITY_INBAND_SECURITY_TLS = 1

// MS_INDICATOR
const MS_INDICATOR_MS_NOT_SUPPORTED = 0
const MS_INDICATOR_MS_SUPPORTED = 1

// QUOTA_TYPE
const QUOTA_TYPE_TIME = 0
const QUOTA_TYPE_MONEY = 1
const QUOTA_TYPE_TOTAL_OCTETS = 2
const QUOTA_TYPE_INPUT_OCTETS = 3
const QUOTA_TYPE_OUTPUT_OCTETS = 4
const QUOTA_TYPE_SERVICE_SPECIFIC_UNITS = 5

// REPORTING_REASON
const REPORTING_REASON_NO_REASON = 0
const REPORTING_REASON_QHT = 1
const REPORTING_REASON_FINAL = 2
const REPORTING_REASON_QUOTA_EXHAUSTED = 3
const REPORTING_REASON_VALIDITY_YIME = 4

// RESULT_CODE
const RESULT_CODE_SUCCESS = 2001
const RESULT_CODE_CREDIT_LIMIT_REACHED = 4012

// APPL_ID
const APPL_ID_DIAMCM = 0 // Diameter Common Messages (RFC3588-11.2.2)
const APPL_ID_NASREQ = 1 // NASREQ					(RFC3588-11.2.2)
const APPL_ID_DIAMIP = 2 // Mobile-IP				(RFC3588-11.2.2)
const APPL_ID_DIAMBA = 3 // Diameter Base Acounting	(RFC3588-11.2.2)
const APPL_ID_DIAMCC = 4 // Diameter Credit Control	(RFC4006-12.1)

const (
	// acronyms for Diameter Answer commands
	DIAMETER_COMMAND_CEA_ACRONYM = "CEA"
	DIAMETER_COMMAND_DWA_ACRONYM = "DWA"
	DIAMETER_COMMAND_DPA_ACRONYM = "DPA"
	DIAMETER_COMMAND_ACA_ACRONYM = "ACA"
	DIAMETER_COMMAND_RAA_ACRONYM = "RAA"
	DIAMETER_COMMAND_ASA_ACRONYM = "ASA"
	DIAMETER_COMMAND_STA_ACRONYM = "STA"
	DIAMETER_COMMAND_CCA_ACRONYM = "CCA"

	// result codes (enum for result-code AVP)
	DIAMETER_RESULT_MULTI_ROUND_AUTH          = 1001
	DIAMETER_RESULT_SUCCESS                   = 2001
	DIAMETER_RESULT_LIMITED_SUCCESS           = 2002
	DIAMETER_RESULT_CONTEXT_ONLY              = 2103
	DIAMETER_RESULT_COMMAND_UNSUPPORTED       = 3001
	DIAMETER_RESULT_UNABLE_TO_DELIVER         = 3002
	DIAMETER_RESULT_REALM_NOT_SERVED          = 3003
	DIAMETER_RESULT_TOO_BUSY                  = 3004
	DIAMETER_RESULT_LOOP_DETECTED             = 3005
	DIAMETER_RESULT_REDIRECT_INDICATION       = 3006
	DIAMETER_RESULT_APPLICATION_UNSUPPORTED   = 3007
	DIAMETER_RESULT_INVALID_HDR_BITS          = 3008
	DIAMETER_RESULT_INVALID_AVP_BITS          = 3009
	DIAMETER_RESULT_UNKNOWN_PEER              = 3010
	DIAMETER_RESULT_AUTHENTICATION_REJECTED   = 4001
	DIAMETER_RESULT_OUT_OF_SPACE              = 4002
	DIAMETER_RESULT_ELECTION_LOST             = 4003
	DIAMETER_RESULT_AVP_UNSUPPORTED           = 5001
	DIAMETER_RESULT_UNKNOWN_SESSION_ID        = 5002
	DIAMETER_RESULT_AUTHORIZATION_REJECTED    = 5003
	DIAMETER_RESULT_INVALID_AVP_VALUE         = 5004
	DIAMETER_RESULT_MISSING_AVP               = 5005
	DIAMETER_RESULT_RESOURCES_EXCEEDED        = 5006
	DIAMETER_RESULT_CONTRADICTING_AVPS        = 5007
	DIAMETER_RESULT_AVP_NOT_ALLOWED           = 5008
	DIAMETER_RESULT_AVP_OCCURS_TOO_MANY_TIMES = 5009
	DIAMETER_RESULT_NO_COMMON_APPLICATION     = 5010
	DIAMETER_RESULT_UNSUPPORTED_VERSION       = 5011
	DIAMETER_RESULT_UNABLE_TO_COMPLY          = 5012
	DIAMETER_RESULT_INVALID_BIT_IN_HEADER     = 5013
	DIAMETER_RESULT_INVALID_AVP_LENGTH        = 5014
	DIAMETER_RESULT_INVALID_MESSAGE_LENGTH    = 5015
	DIAMETER_RESULT_INVALID_AVP_BIT_COMBO     = 5016
	DIAMETER_RESULT_NO_COMMON_SECURITY        = 5017
)

type DiamCH struct {
	TaskName string
	Message  *diam.Message
}

// Формирование сообщения
func CreateCCREventMessage(Msisdn RecTypePool, date time.Time, RecordType RecTypeRatioType, dict *dict.Parser) (*diam.Message, error) {
	sid := "session;" + strconv.Itoa(int(rand.Uint32()))
	diam_message := diam.NewRequest(COMMAND_CODE_Credit_Control, 4, dict)
	diam_message.NewAVP(avp.SessionID, avp.Mbit, 0, datatype.UTF8String(sid))

	//{ Auth-Application-Id }
	diam_message.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(4))
	//{ Service-Context-Id } из конфига БРТ для каждого лоджик кола
	diam_message.NewAVP(avp.ServiceContextID, avp.Mbit, 0, datatype.UTF8String("internet.volume.pcef.vpcef"))
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
			diam.NewAVP(avp.SubscriptionIDData, avp.Mbit, 0, datatype.UTF8String("79251470282")),
		},
	})
	diam_message.NewAVP(avp.SubscriptionID, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.SubscriptionIDType, avp.Mbit, 0, datatype.Enumerated(1)),
			diam.NewAVP(avp.SubscriptionIDData, avp.Mbit, 0, datatype.UTF8String("250020153589056")),
		},
	})
	diam_message.NewAVP(avp.UserName, avp.Mbit, 0, datatype.UTF8String("79251470282"))
	//{ Event-Timestamp }  Время события
	diam_message.NewAVP(avp.EventTimestamp, avp.Mbit, 0, datatype.Time(time.Now()))
	//{ Multiple-Services-Indicator } Нужен?
	diam_message.NewAVP(avp.MultipleServicesIndicator, avp.Mbit, 0, datatype.Enumerated(1))
	// { Requested-Action } Безусловное списание 0
	diam_message.NewAVP(avp.RequestedAction, avp.Mbit, 0, datatype.Enumerated(0))
	// { Multiple-Services-Credit-Control }
	diam_message.NewAVP(avp.MultipleServicesCreditControl, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			// Requested-Service-Unit
			diam.NewAVP(avp.RequestedServiceUnit, avp.Mbit, 0, &diam.GroupedAVP{
				AVP: []*diam.AVP{
					diam.NewAVP(avp.CCTime, avp.Mbit, 0, datatype.Unsigned32(rand.Intn(100))),
				},
			}),
		},
	})

	// { Service-Information }
	/*diam_message.NewAVP(avp.ServiceInformation, avp.Mbit, 10415, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.PSInformation, avp.Mbit, 10415, &diam.GroupedAVP{
				AVP: []*diam.AVP{
					diam.NewAVP(avp.CallingStationID, avp.Mbit, 0, datatype.UTF8String("1dfgd")),
				},
			}),
		},
	})*/
	return diam_message, nil
}

// Обработчик ответа
func ResponseDiamHandler(message *diam.Message, log *log.Logger, debug bool) {

	op := ""
	cmd, err := message.Dictionary().FindCommand(
		message.Header.ApplicationID,
		message.Header.CommandCode,
	)
	if err != nil {
		op += "unknown_command"
	} else {
		op += cmd.Short + "A"
	}

	// выделение кода ответа
	mm, _ := message.FindAVPs(268, 0)
	//var diamcode string
	ans := "DIAM: Answer " + op + " code:"

	for _, i := range mm {
		ans += ConvertType(i) + ";"
	}

	log.Println(ans)
	// Текст ошибки
	mm, _ = message.FindAVPs(avp.ErrorMessage, 0)
	for _, i := range mm {
		log.Println("ERROR: " + ConvertType(i))
	}
}

// Конвертер для ошибок в строку
func ConvertType(m *diam.AVP) string {
	switch m.Data.Type() {
	case 16:
		replacer := strings.NewReplacer("Unsigned32{", "", "}", "")
		return replacer.Replace(m.Data.String())
	default:
		return m.Data.String()
	}

}
