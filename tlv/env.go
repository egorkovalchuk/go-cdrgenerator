package tlv

const (
	// Тип сообщения на отправку приветственного сообщения (инициатор)
	// Идет с BRT
	TYPE_STARTUP_REQ uint16 = 0x0001
	// Тип сообщения на отправку приветственного сообщения (подтверждение)
	// Идет с SCP -> BRT ответ
	TYPE_STARTUP_RESP uint16 = 0x0002
	// Тип сообщения на отправку подтверждения жизни (инициатор)
	TYPE_KEEPALIVE_REQ uint16 = 0x0007
	// Тип сообщения на отправку подтверждения жизни (подтверждение)
	TYPE_KEEPALIVE_RESP uint16 = 0x0008
	// Тип сообщения на отправку авторизации вызова (инициатор)
	TYPE_AUTHORIZEVOICE_REQ uint16 = 0x1001
	// Тип сообщения на отправку реавторизаци вызова (запрос на дополнительную квоту) (инициатор)
	TYPE_REAUTHORIZEVOICE_REQ uint16 = 0x1002
	// Тип сообщения на подтверждение авторизации вызова (подтверждение)
	TYPE_AUTHORIZEVOICE_CONFIRM uint16 = 0x1003
	// Тип сообщения на отказ авторизации вызова (подтверждение)
	TYPE_AUTHORIZEVOICE_REJECT uint16 = 0x1004
	// Тип сообщения на завершение звонка (инициатор)
	TYPE_ENDVOICE_REQ uint16 = 0x1005
	// Тип сообщения на завершение звонка (подтверждение)
	TYPE_ENDVOICE_RESP uint16 = 0x1006
	// Тип сообщения на отправку авторизации sms (инициатор)
	TYPE_AUTHORIZESMS_REQ uint16 = 0x3001
	// Тип сообщения на подтверждение авторизации sms (подтверждение)
	TYPE_AUTHORIZESMS_CONFIRM uint16 = 0x3002
	// Тип сообщения на отказ авторизации sms (подтверждение)
	TYPE_AUTHORIZESMS_REJECT uint16 = 0x3003
	// Тип сообщения на завершение sms (инициатор)
	TYPE_ENDSMS_REQ uint16 = 0x3004
	// Тип сообщения на завершение sms (подтверждение)
	TYPE_ENDSMS_RESP uint16 = 0x3005
)

var camel_params_desc = []camel_param_desc{
	{Name: "AccessPointName", Type: "string", MaxLen: 50, Tag: 0x0001, Optional: false},
	{Name: "AdditionalCallingPartyNumber", Type: "string", MaxLen: 64, Tag: 0x0002, Optional: false},
	{Name: "AdditionalCallingPartyNumberInformation", Type: "buffer", MaxLen: 3, Tag: 0x0003, Optional: false},
	{Name: "CallAttemptElapsedTime", Type: "integer", MaxLen: 4, Tag: 0x0004, Optional: false},
	{Name: "CallConnectedElapsedTime", Type: "integer", MaxLen: 4, Tag: 0x0005, Optional: false},
	{Name: "CalledPartyBCDInformation", Type: "buffer", MaxLen: 3, Tag: 0x0006, Optional: false},
	{Name: "CalledPartyBCDNumber", Type: "string", MaxLen: 64, Tag: 0x0007, Optional: false},
	{Name: "CalledPartyNumber", Type: "string", MaxLen: 64, Tag: 0x0008, Optional: false},
	{Name: "CalledPartyNumberInformation", Type: "buffer", MaxLen: 3, Tag: 0x0009, Optional: false},
	{Name: "CallingPartyNumber", Type: "string", MaxLen: 64, Tag: 0x000A, Optional: false},
	{Name: "CallingPartyNumberInformation", Type: "buffer", MaxLen: 3, Tag: 0x000B, Optional: false},
	{Name: "CallingPartysCategory", Type: "byte", MaxLen: 1, Tag: 0x000C, Optional: false},
	{Name: "CallReferenceNumber", Type: "buffer", MaxLen: 8, Tag: 0x000D, Optional: false},
	{Name: "CallStopTime", Type: "string", MaxLen: 20, Tag: 0x000E, Optional: false},
	{Name: "ConversationTime", Type: "integer", MaxLen: 4, Tag: 0x000F, Optional: false},
	{Name: "DestinationRoutingNumber", Type: "string", MaxLen: 64, Tag: 0x0010, Optional: false},
	{Name: "DestinationRoutingNumberInformation", Type: "buffer", MaxLen: 3, Tag: 0x0011, Optional: false},
	{Name: "Disconnect", Type: "boolean", MaxLen: 1, Tag: 0x0012, Optional: false},
	{Name: "ErrorCode", Type: "integer", MaxLen: 4, Tag: 0x0013, Optional: false},
	{Name: "EventTypeBCSM", Type: "byte", MaxLen: 1, Tag: 0x0014, Optional: false},
	{Name: "EventTypeSMS", Type: "byte", MaxLen: 1, Tag: 0x0015, Optional: false},
	{Name: "GPRSEventType", Type: "byte", MaxLen: 1, Tag: 0x0016, Optional: false},
	{Name: "IMSI", Type: "string", MaxLen: 15, Tag: 0x0017, Optional: false},
	{Name: "LocationInformation", Type: "buffer", MaxLen: 1, Tag: 0x0018, Optional: false},
	{Name: "LocationInformationMSC", Type: "buffer", MaxLen: 256, Tag: 0x0019, Optional: false},
	{Name: "LocationNumber", Type: "string", MaxLen: 64, Tag: 0x001A, Optional: false},
	{Name: "LocationNumberInformation", Type: "buffer", MaxLen: 3, Tag: 0x001B, Optional: false},
	{Name: "MaxVolume", Type: "integer", MaxLen: 4, Tag: 0x001C, Optional: false},
	{Name: "MscAddressInformation", Type: "buffer", MaxLen: 3, Tag: 0x001D, Optional: false},
	{Name: "MscAddressNumber", Type: "string", MaxLen: 64, Tag: 0x001E, Optional: false},
	{Name: "MSISDNInformation", Type: "buffer", MaxLen: 3, Tag: 0x001F, Optional: false},
	{Name: "MSISDNNumber", Type: "string", MaxLen: 15, Tag: 0x0020, Optional: false},
	{Name: "OriginalCalledInformation", Type: "buffer", MaxLen: 3, Tag: 0x0021, Optional: false},
	{Name: "OriginalCalledNumber", Type: "string", MaxLen: 64, Tag: 0x0022, Optional: false},
	{Name: "ProtocolVersion", Type: "string", MaxLen: 10, Tag: 0x0023, Optional: false},
	{Name: "RedirectingNumber", Type: "string", MaxLen: 64, Tag: 0x0024, Optional: false},
	{Name: "RedirectingNumberInformation", Type: "buffer", MaxLen: 3, Tag: 0x0025, Optional: false},
	{Name: "ReleaseCause", Type: "integer", MaxLen: 4, Tag: 0x0026, Optional: false},
	{Name: "ReleaseCauseGPRS", Type: "integer", MaxLen: 4, Tag: 0x0027, Optional: false},
	{Name: "ReleaseCauseSMS", Type: "integer", MaxLen: 4, Tag: 0x0028, Optional: false},
	{Name: "ServerSubscribers", Type: "string", MaxLen: 1024, Tag: 0x0029, Optional: false},
	{Name: "ServiceCode", Type: "integer", MaxLen: 4, Tag: 0x002A, Optional: false},
	{Name: "ServiceKey", Type: "integer", MaxLen: 4, Tag: 0x002B, Optional: false},
	{Name: "SessionID", Type: "string", MaxLen: 64, Tag: 0x002C, Optional: false},
	{Name: "SMSCAddressInformation", Type: "buffer", MaxLen: 3, Tag: 0x002D, Optional: false},
	{Name: "SMSCAddressNumber", Type: "string", MaxLen: 64, Tag: 0x002E, Optional: false},
	{Name: "SMSStatus", Type: "byte", MaxLen: 1, Tag: 0x002F, Optional: false},
	{Name: "TimeAndTimezone", Type: "buffer", MaxLen: 8, Tag: 0x0030, Optional: false},
	{Name: "TransferredVolume", Type: "integer", MaxLen: 4, Tag: 0x0031, Optional: false},
	{Name: "GPRSTrafficType", Type: "byte", MaxLen: 1, Tag: 0x0032, Optional: false},
	{Name: "GGSNAddress", Type: "buffer", MaxLen: 17, Tag: 0x0033, Optional: false},
	{Name: "SGSNAddressNumber", Type: "string", MaxLen: 64, Tag: 0x0034, Optional: false},
	{Name: "SGSNAddressInformation", Type: "buffer", MaxLen: 3, Tag: 0x0035, Optional: false},
	{Name: "LocationInformationGPRS", Type: "buffer", MaxLen: 1, Tag: 0x0036, Optional: false},
	{Name: "VLRAddressNumber", Type: "string", MaxLen: 64, Tag: 0x0037, Optional: false},
	{Name: "VLRAddressInformation", Type: "buffer", MaxLen: 3, Tag: 0x0038, Optional: false},
	{Name: "RedirectingInformation", Type: "buffer", MaxLen: 8, Tag: 0x0039, Optional: false},
	{Name: "EndReason", Type: "byte", MaxLen: 1, Tag: 0x003A, Optional: false},
	{Name: "SSEvent", Type: "byte", MaxLen: 1, Tag: 0x003B, Optional: false},
	{Name: "SSEventSpecification", Type: "buffer", MaxLen: 2, Tag: 0x003C, Optional: false},
	{Name: "ForwardConferenceTreatmentIndicator", Type: "byte", MaxLen: 1, Tag: 0x003D, Optional: false},
	{Name: "BackwardConferenceTreatmentIndicator", Type: "byte", MaxLen: 1, Tag: 0x003E, Optional: false},
	{Name: "ServiceCodeType", Type: "byte", MaxLen: 1, Tag: 0x003F, Optional: false},
	{Name: "Charge", Type: "boolean", MaxLen: 1, Tag: 0x0040, Optional: false},
	{Name: "FurnishChargingInformation", Type: "boolean", MaxLen: 1, Tag: 0x0041, Optional: false},
	{Name: "CAPProtocolVersion", Type: "byte", MaxLen: 1, Tag: 0x0042, Optional: false},
	{Name: "CUGIndex", Type: "Short", MaxLen: 2, Tag: 0x0043, Optional: false},
	{Name: "CUGInterlock", Type: "string", MaxLen: 4, Tag: 0x0044, Optional: false},
	{Name: "CUGOutgoingAccess", Type: "byte", MaxLen: 1, Tag: 0x0045, Optional: false},
	{Name: "BRTIntermalID", Type: "byte", MaxLen: 1, Tag: 0x0050, Optional: false},
	{Name: "GSMForwardingPending", Type: "byte", MaxLen: 1, Tag: 0x0051, Optional: false},
	{Name: "ToneID", Type: "integer", MaxLen: 1, Tag: 0x0052, Optional: false},
	{Name: "ToneDuration", Type: "integer", MaxLen: 1, Tag: 0x0053, Optional: false},
	{Name: "ServiceInteractionIndicatorsTwo", Type: "buffer", MaxLen: 1, Tag: 0x0060, Optional: false},
	{Name: "SubscriberState", Type: "byte", MaxLen: 1, Tag: 0x0061, Optional: false},
	{Name: "gmscAddress", Type: "string", MaxLen: 64, Tag: 0x0062, Optional: false},
	{Name: "gmscAddressInformation", Type: "buffer", MaxLen: 3, Tag: 0x0063, Optional: false},
	{Name: "IMEI", Type: "string", MaxLen: 16, Tag: 0x0064, Optional: false},
	{Name: "MaxElapsedTime", Type: "integer", MaxLen: 4, Tag: 0x0065, Optional: false},
	{Name: "UserData", Type: "string", MaxLen: 256, Tag: 0x0066, Optional: false},
	{Name: "AdditionalInformation", Type: "buffer", MaxLen: 1, Tag: 0x0067, Optional: false},
}

var camel_type = []camel_type_len{
	{Type: "integer", MaxLen: 4, Static: true},
	{Type: "boolean", MaxLen: 1, Static: true},
	{Type: "string", MaxLen: 65535, Static: false},
	{Type: "buffer", MaxLen: 65535, Static: false},
	{Type: "byte", MaxLen: 1, Static: true},
	{Type: "short", MaxLen: 2, Static: true},
}
