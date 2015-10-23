// Copyright 2015 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style license described in the
// LICENSE file.

// Package contains IPMI 2.0 spec implementation
package ipmigod

import (
	"encoding/binary"
	"fmt"
	_ "github.com/platinasystems/goes/cli"
	"log"
	"net"
)

const (
	MAX_MSG_RETURN_DATA = 1000
)

func ipmiLanInit() {

	// Initialize user database for straight authentication
	lanserv.users[1].idx = 1
	lanserv.users[1].username = make([]uint8, 16)
	copy(lanserv.users[1].username[0:], "")
	lanserv.users[1].pw = make([]uint8, 16)
	copy(lanserv.users[1].pw[0:], "test")
	lanserv.users[1].maxPriv = IPMI_PRIVILEGE_USER
	lanserv.users[1].allowedAuths = (1 << IPMI_AUTHTYPE_NONE) |
		//(1 << AUTHTYPE_MD2) |
		//(1 << AUTHTYPE_MD5) |
		(1 << IPMI_AUTHTYPE_STRAIGHT)
	lanserv.users[1].valid = true
	lanserv.users[2].idx = 2
	lanserv.users[2].username = make([]uint8, 16)
	copy(lanserv.users[2].username[0:], "ipmiusr")
	lanserv.users[2].pw = make([]uint8, 16)
	copy(lanserv.users[2].pw[0:], "test")
	lanserv.users[2].maxPriv = IPMI_PRIVILEGE_ADMIN
	lanserv.users[2].allowedAuths = (1 << IPMI_AUTHTYPE_NONE) //|
	//(1 << AUTHTYPE_MD2) |
	//(1 << AUTHTYPE_MD5) |
	//(1 << IPMI_AUTHTYPE_STRAIGHT)
	lanserv.users[2].valid = true

	lanserv.chanNum = 1
	lanserv.defaultSessionTimeout = 30
	lanserv.sidSeq = 0
	lanserv.nextChallSeq = 0
	lanserv.chanPrivLimit = IPMI_PRIVILEGE_ADMIN
	lanserv.chanPrivAllowedAuths[IPMI_PRIVILEGE_CALLBACK-1] =
		(1 << IPMI_AUTHTYPE_MD5)
	lanserv.chanPrivAllowedAuths[IPMI_PRIVILEGE_USER-1] =
		(1 << IPMI_AUTHTYPE_NONE)
	lanserv.chanPrivAllowedAuths[IPMI_PRIVILEGE_OPERATOR-1] =
		(1 << IPMI_AUTHTYPE_NONE)
	lanserv.chanPrivAllowedAuths[IPMI_PRIVILEGE_ADMIN-1] =
		(1 << IPMI_AUTHTYPE_STRAIGHT)
	lanserv.chanPrivAllowedAuths[IPMI_PRIVILEGE_OEM-1] =
		(1 << IPMI_AUTHTYPE_OEM)

	for i := 1; i < MAX_SESSIONS+1; i++ {
		lanserv.sessions[i].handle = uint32(i)
	}

}

func init() {
	// daemon setup
	// Do startup initialization for daemon
	//  - replaces lan config file and emu config file
	// Initialize channels[1] as lan channel
	// Initialize following
	// addr :: 623
	// priv_limit admin
	// allowed_auths_callback none md2 md5 straight
	// allowed_auths_user none md2 md5 straight
	// allowed_auths_operator none md2 md5 straight
	// allowed_auths_admin none md2 md5 straight
	// guid a123456789abcdefa123456789abcdef
	//  user 2 true  "ipmiusr" "test" admin    10 none md2 md5 straight
	ipmiLanInit()

	// Initialize BMC SDRs/Sensors
	bmcInit()

	// Initialize persistence database
}

func Main() {

	// Listen on UDP port 623 on all interfaces.
	serverAddr, err := net.ResolveUDPAddr("udp", ":623")
	if err != nil {
		log.Fatal(err)
	}

	// Now listen at selected port
	serverConn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer serverConn.Close()

	for {

		msg := new(msgT)
		n, remoteAddr, err :=
			serverConn.ReadFromUDP(msg.data[0:])
		msg.remoteAddr = remoteAddr
		if debug {
			fmt.Println("Received ", n, " bytes from ",
				msg.remoteAddr)
		}
		if err != nil {
			fmt.Println("Error: ", err)
			fmt.Printf("Error: Received %d bytes\n", n)
		}
		msg.dataLen = uint(n)
		msg.conn = serverConn
		msg.ipmiHandleMsg()
	}
}

func (msg *msgT) ipmiHandleMsg() {

	if msg.dataLen < 5 {
		fmt.Printf("LAN msg failure: message too short %d",
			msg.dataLen)
		return
	}
	msg.channel = lanserv.chanNum

	// Parse incoming IPMI packet (including error checks)
	// and load up msg struct
	msg.ipmiParseMsg()

	if msg.authtype == IPMI_AUTHTYPE_RMCP_PLUS {
		//ipmi_handle_rmcpp_msg(lan, &msg);
		fmt.Println("Received RMCP+ message!")
	} else {
		if debug {
			fmt.Println("Received RMCP message!")
		}
		(netfuncProcessors[msg.rmcp.message.netfn])(msg)
	}
}

func (msg *msgT) ipmiParseMsg() {
	dataStart := msg.dataStart

	if msg.data[dataStart+3] == 6 {
		// Handle ASF ping message
		asfPing(msg)
	} else if msg.data[dataStart+3] == 7 {
		// Peek ahead to see if we have an RMCP or RMCP+ message
		if msg.data[dataStart+4] == IPMI_AUTHTYPE_RMCP_PLUS {
			fmt.Println("LAN msg not supported RMCP+")
			//ipmi_parse_rmcpp_msg(msg)
		} else {
			msg.ipmiParseRmcpMsg()
		}
	} else {
		fmt.Println("LAN msg has unsupported class",
			msg.data[dataStart+3])
	}
}

func (msg *msgT) ipmiParseRmcpMsg() {
	dataStart := msg.dataStart

	// Load RMCP header
	msg.rmcp.hdr.version = msg.data[dataStart+0]
	msg.rmcp.hdr.rmcpSeq = msg.data[dataStart+2]
	msg.rmcp.hdr.class = msg.data[dataStart+3]
	msg.dataStart += 4
	dataStart = msg.dataStart

	if msg.rmcp.hdr.rmcpSeq != 0xff {
		fmt.Println("LAN msg failure: seq not ff")
		return /* Sequence # must be ff (no ack) */
	}

	// Load IPMI Session fields
	msg.rmcp.session.authType = msg.data[dataStart]
	msg.authtype = msg.rmcp.session.authType
	msg.rmcp.session.seq =
		binary.LittleEndian.Uint32(msg.data[dataStart+1 : dataStart+5])
	msg.rmcp.session.sid =
		binary.LittleEndian.Uint32(msg.data[dataStart+5 : dataStart+9])
	if debug {
		fmt.Printf("Session_id from freeipmi: %x\n",
			msg.rmcp.session.sid)
	}
	msg.sid = msg.rmcp.session.sid
	if msg.rmcp.session.authType != IPMI_AUTHTYPE_NONE {
		copy(msg.rmcp.session.authCode[0:],
			msg.data[dataStart+9:dataStart+25])
		msg.rmcp.session.payloadLgth = msg.data[dataStart+25]
		msg.dataStart += 26
	} else {
		msg.dataStart += 10
	}
	dataStart = msg.dataStart

	// Load IPMI Message fields
	msg.rmcp.message.rsAddr = msg.data[dataStart]
	msg.rmcp.message.netfn = msg.data[dataStart+1] >> 2
	msg.rmcp.message.rsLun = msg.data[dataStart+1] & 0x3
	msg.rmcp.message.rqAddr = msg.data[dataStart+3]
	msg.rmcp.message.rqSeq = msg.data[dataStart+4] >> 2
	msg.rmcp.message.rqLun = msg.data[dataStart+4] & 0x3
	msg.rmcp.message.cmd = msg.data[dataStart+5]
	msg.dataStart += 6
}

func (msg *msgT) returnRsp(session *sessionT, rsp *rspMsgDataT) {
	var (
		data         [MAX_MSG_RETURN_DATA]uint8
		csum         int8
		dummySession sessionT
		len          int
	)

	if session == nil {
		session = sidToSession(msg.sid)
	}
	if session != nil && session.rmcpplus {
		//rmcp plus not currently supported
		fmt.Println("RMCP+ returnRsp not supported!")
		return
	} else if msg.sid == 0 {
		session = &dummySession
		session.active = true
		session.authtype = IPMI_AUTHTYPE_NONE
		session.xmitSeq = 0
		session.sid = 0
	}

	if session == nil {
		fmt.Println("returnRsp: Can't find session")
		return
	}

	// Build the return packet
	dcur := 0
	data[dcur] = 6 /* RMCP version. */
	dcur++
	data[dcur] = 0
	dcur++
	data[dcur] = 0xff /* No seq num */
	dcur++
	data[dcur] = 7 /* IPMI msg class */
	dcur++
	data[dcur] = session.authtype
	dcur++
	binary.LittleEndian.PutUint32(data[dcur:dcur+4], session.xmitSeq)
	session.xmitSeq++
	if session.xmitSeq == 0 {
		session.xmitSeq++
	}
	dcur += 4
	binary.LittleEndian.PutUint32(data[dcur:dcur+4], session.sid)
	dcur += 4
	if session.authtype != IPMI_AUTHTYPE_NONE {
		dcur += 16 // sizeof rmcp.session.auth_code[]
	}
	// Add message structure length to specified payload length
	len = int(rsp.dataLen + 7) // rmcp.message layer size
	data[dcur] = uint8(len)
	dcur++
	startOfMsg := dcur
	data[dcur] = msg.rmcp.message.rqAddr
	dcur++
	data[dcur] = (rsp.netfn << 2) | msg.rmcp.message.rqLun
	dcur++
	data[dcur] = uint8(ipmiChecksum(data[startOfMsg:startOfMsg+2], 2, 0))
	if debug {
		fmt.Printf("csum1: %x\n", data[dcur])
	}
	dcur++
	data[dcur] = msg.rmcp.message.rsAddr
	dcur++
	data[dcur] = (msg.rmcp.message.rqSeq << 2) | msg.rmcp.message.rsLun
	dcur++
	data[dcur] = rsp.cmd
	dcur++
	// copy the response payload data into msg data
	copy(data[dcur:], rsp.data[0:rsp.dataLen])
	csum = -ipmiChecksum(data[dcur-3:dcur], 3, 0)
	csum = ipmiChecksum(data[dcur:dcur+int(rsp.dataLen)],
		int(rsp.dataLen), csum)
	dcur += int(rsp.dataLen)
	data[dcur] = uint8(csum)
	if debug {
		fmt.Printf("csum2: %x\n", data[dcur])
	}
	dcur++
	if session.authtype != IPMI_AUTHTYPE_NONE {
		// authgen needed for real authtype
		//rv = auth_gen(session, data+13,
		//	data+9, data+5,
		//	pos, 6,
		//    rsp->data, rsp->data_len,
		//    &csum, 1);
	}
	if debug {
		fmt.Println("Sending", dcur, " bytes to", msg.remoteAddr)
	}
	msg.conn.WriteToUDP(data[0:dcur], msg.remoteAddr)
}

func (msg *msgT) returnErr(session *sessionT, err uint8) {

	var rsp rspMsgDataT

	rsp.netfn = msg.rmcp.message.netfn | 1
	rsp.cmd = msg.rmcp.message.cmd
	rsp.data[0] = err
	rsp.dataLen = 1
	msg.returnRsp(session, &rsp)
}

func (msg *msgT) returnRspData(session *sessionT, data []uint8,
	dataLength uint) {
	var rsp rspMsgDataT

	rsp.netfn = msg.rmcp.message.netfn | 1
	rsp.cmd = msg.rmcp.message.cmd
	copy(rsp.data[0:], data[0:dataLength])
	rsp.dataLen = uint16(dataLength)

	msg.returnRsp(session, &rsp)
}

type ipmiNetfuncProcessor func(*msgT)

var netfuncProcessors = map[uint8]ipmiNetfuncProcessor{
	CHASSIS_NETFN:         chassisNetfn,
	BRIDGE_NETFN:          bridgeNetfn,
	SENSOR_EVENT_NETFN:    sensorEventNetfn,
	APP_NETFN:             appNetfn,
	FIRMWARE_NETFN:        firmwareNetfn,
	STORAGE_NETFN:         storageNetfn,
	TRANSPORT_NETFN:       transportNetfn,
	GROUP_EXTENSION_NETFN: groupExtensionNetfn,
	OEM_GROUP_NETFN:       oemGroupNetfn,
}

type chassisProcessor func(*msgT)

var chassisProcessors = map[uint8]chassisProcessor{
	GET_CHASSIS_CAPABILITIES_CMD: getChassisCapabilities,
	CHASSIS_CONTROL_CMD:          chassisControl,
	CHASSIS_RESET_CMD:            chassisReset,
	CHASSIS_IDENTIFY_CMD:         chassisIdentify,
	SET_CHASSIS_CAPABILITIES_CMD: setChassisCapabilities,
	SET_POWER_RESTORE_POLICY_CMD: setPowerRestorePolicy,
	GET_SYSTEM_RESTART_CAUSE_CMD: getSystemRestartCause,
	SET_SYSTEM_BOOT_OPTIONS_CMD:  setSystemBootOptions,
	GET_SYSTEM_BOOT_OPTIONS_CMD:  getSystemBootOptions,
}

func chassisNetfn(msg *msgT) {
	(chassisProcessors[msg.rmcp.message.cmd])(msg)
}

type bridgeProcessor func(*msgT)

var bridgeProcessors = map[uint8]bridgeProcessor{
	GET_BRIDGE_STATE_CMD:         getBridgeState,
	SET_BRIDGE_STATE_CMD:         setBridgeState,
	GET_ICMB_ADDRESS_CMD:         getIcmbAddress,
	SET_ICMB_ADDRESS_CMD:         setIcmbAddress,
	SET_BRIDGE_PROXY_ADDRESS_CMD: setBridgeProxyAddress,
	GET_BRIDGE_STATISTICS_CMD:    getBridgeStatistics,
	GET_ICMB_CAPABILITIES_CMD:    getIcmbCapabilities,

	CLEAR_BRIDGE_STATISTICS_CMD:  clearBridgeStatistics,
	GET_BRIDGE_PROXY_ADDRESS_CMD: getBridgeProxyAddress,
	GET_ICMB_CONNECTOR_INFO_CMD:  getIcmbConnectorInfo,
	SET_ICMB_CONNECTOR_INFO_CMD:  setIcmbConnectorInfo,
	SEND_ICMB_CONNECTION_ID_CMD:  sendIcmbConnectionId,

	PREPARE_FOR_DISCOVERY_CMD: prepareForDiscovery,
	GET_ADDRESSES_CMD:         getAddresses,
	SET_DISCOVERED_CMD:        setDiscovered,
	GET_CHASSIS_DEVICE_ID_CMD: getChassisDeviceId,
	SET_CHASSIS_DEVICE_ID_CMD: setChassisDeviceId,

	BRIDGE_REQUEST_CMD: bridgeRequest,
	BRIDGE_MESSAGE_CMD: bridgeMessage,

	GET_EVENT_COUNT_CMD:           getEventCount,
	SET_EVENT_DESTINATION_CMD:     setEventDestination,
	SET_EVENT_RECEPTION_STATE_CMD: setEventReceptionState,
	SEND_ICMB_EVENT_MESSAGE_CMD:   sendIcmbEventMessage,
	GET_EVENT_DESTIATION_CMD:      getEventDestination,
	GET_EVENT_RECEPTION_STATE_CMD: getEventReceptionState,

	ERROR_REPORT_CMD: errorReport,
}

func bridgeNetfn(msg *msgT) {
	fmt.Println("bridgeNetfn not supported",
		msg.rmcp.message.cmd)
}

type sensorProcessor func(*msgT)

var sensorProcessors = map[uint8]sensorProcessor{
	SET_EVENT_RECEIVER_CMD: setEventReceiver,
	GET_EVENT_RECEIVER_CMD: getEventReceiver,
	PLATFORM_EVENT_CMD:     platformEvent,

	GET_PEF_CAPABILITIES_CMD:        getPefCapabilities,
	ARM_PEF_POSTPONE_TIMER_CMD:      armPefPostponeTimer,
	SET_PEF_CONFIG_PARMS_CMD:        setPefConfigParms,
	GET_PEF_CONFIG_PARMS_CMD:        getPefConfigParms,
	SET_LAST_PROCESSED_EVENT_ID_CMD: setLastProcessedEventId,
	GET_LAST_PROCESSED_EVENT_ID_CMD: getLastProcessedEventId,
	ALERT_IMMEDIATE_CMD:             alertImmediate,
	PET_ACKNOWLEDGE_CMD:             petAcknowledge,

	GET_DEVICE_SDR_INFO_CMD:           getDeviceSdrInfo,
	GET_DEVICE_SDR_CMD:                getDeviceSdr,
	RESERVE_DEVICE_SDR_REPOSITORY_CMD: reserveDeviceSdrRepository,
	GET_SENSOR_READING_FACTORS_CMD:    getSensorReadingFactors,
	SET_SENSOR_HYSTERESIS_CMD:         setSensorHysteresis,
	GET_SENSOR_HYSTERESIS_CMD:         getSensorHysteresis,
	SET_SENSOR_THRESHOLD_CMD:          setSensorThreshold,
	GET_SENSOR_THRESHOLD_CMD:          getSensorThreshold,
	SET_SENSOR_EVENT_ENABLE_CMD:       setSensorEventEnable,
	GET_SENSOR_EVENT_ENABLE_CMD:       getSensorEventEnable,
	REARM_SENSOR_EVENTS_CMD:           rearmSensorEvents,
	GET_SENSOR_EVENT_STATUS_CMD:       getSensorEventStatus,
	GET_SENSOR_READING_CMD:            getSensorReading,
	SET_SENSOR_TYPE_CMD:               setSensorType,
	GET_SENSOR_TYPE_CMD:               getSensorType,
}

func sensorEventNetfn(msg *msgT) {
	(sensorProcessors[msg.rmcp.message.cmd])(msg)
}

type appProcessor func(*msgT)

var appProcessors = map[uint8]appProcessor{
	GET_DEVICE_ID_CMD:                 getDeviceId,
	COLD_RESET_CMD:                    coldReset,
	WARM_RESET_CMD:                    warmReset,
	GET_SELF_TEST_RESULTS_CMD:         getSelfTestResults,
	MANUFACTURING_TEST_ON_CMD:         manufacturingTestOn,
	SET_ACPI_POWER_STATE_CMD:          setAcpiPowerState,
	GET_ACPI_POWER_STATE_CMD:          getAcpiPowerState,
	GET_DEVICE_GUID_CMD:               getDeviceGuid,
	RESET_WATCHDOG_TIMER_CMD:          resetWatchdogTimer,
	SET_WATCHDOG_TIMER_CMD:            setWatchdogTimer,
	GET_WATCHDOG_TIMER_CMD:            getWatchdogTimer,
	SET_BMC_GLOBAL_ENABLES_CMD:        setBmcGlobalEnables,
	GET_BMC_GLOBAL_ENABLES_CMD:        getBmcGlobalEnables,
	CLEAR_MSG_FLAGS_CMD:               clearMsgFlags,
	GET_MSG_FLAGS_CMD:                 getMsgFlagsCmd,
	ENABLE_MESSAGE_CHANNEL_RCV_CMD:    enableMessageChannelRcv,
	GET_MSG_CMD:                       getMsg,
	SEND_MSG_CMD:                      sendMsg,
	READ_EVENT_MSG_BUFFER_CMD:         readEventMsgBuffer,
	GET_BT_INTERFACE_CAPABILITIES_CMD: getBtInterfaceCapabilties,
	GET_SYSTEM_GUID_CMD:               getSystemGuid,
	GET_CHANNEL_AUTH_CAPABILITIES_CMD: getChannelAuthCapabilties,
	GET_SESSION_CHALLENGE_CMD:         getSessionChallenge,
	ACTIVATE_SESSION_CMD:              activateSession,
	SET_SESSION_PRIVILEGE_CMD:         setSessionPrivilege,
	CLOSE_SESSION_CMD:                 closeSession,
	GET_SESSION_INFO_CMD:              getSessionInfo,

	GET_AUTHCODE_CMD:                  getAuthcode,
	SET_CHANNEL_ACCESS_CMD:            setChannelAccess,
	GET_CHANNEL_ACCESS_CMD:            getChannelAccess,
	GET_CHANNEL_INFO_CMD:              getChannelInfo,
	SET_USER_ACCESS_CMD:               setUserAccess,
	GET_USER_ACCESS_CMD:               getUserAccess,
	SET_USER_NAME_CMD:                 setUserName,
	GET_USER_NAME_CMD:                 getUserName,
	SET_USER_PASSWORD_CMD:             setUserPassword,
	ACTIVATE_PAYLOAD_CMD:              activatePayload,
	DEACTIVATE_PAYLOAD_CMD:            deavtivatePayload,
	GET_PAYLOAD_ACTIVATION_STATUS_CMD: getPayloadActivationStatus,
	GET_PAYLOAD_INSTANCE_INFO_CMD:     getPayloadInstanceInfo,
	SET_USER_PAYLOAD_ACCESS_CMD:       setUserPayloadAccess,
	GET_USER_PAYLOAD_ACCESS_CMD:       getUserPayloadAccess,
	GET_CHANNEL_PAYLOAD_SUPPORT_CMD:   getChannelPayloadSupport,
	GET_CHANNEL_PAYLOAD_VERSION_CMD:   getChannelPayloadVersion,
	GET_CHANNEL_OEM_PAYLOAD_INFO_CMD:  getChannelOemPayloadInfo,

	MASTER_READ_WRITE_CMD: masterReadWrite,

	GET_CHANNEL_CIPHER_SUITES_CMD:         getChannelCipherSuites,
	SUSPEND_RESUME_PAYLOAD_ENCRYPTION_CMD: suspendResumePayloadEncryption,
	SET_CHANNEL_SECURITY_KEY_CMD:          setChannelSecurityKey,
	GET_SYSTEM_INTERFACE_CAPABILITIES_CMD: getSystemInterfaceCapabilities,
}

func appNetfn(msg *msgT) {
	(appProcessors[msg.rmcp.message.cmd])(msg)
}

func firmwareNetfn(msg *msgT) {
	fmt.Println("firmwareNetfn not supported",
		msg.rmcp.message.cmd)
}

type storageProcessor func(*msgT)

var storageProcessors = map[uint8]storageProcessor{
	GET_FRU_INVENTORY_AREA_INFO_CMD: getFruInventoryAreaInfo,
	READ_FRU_DATA_CMD:               readFruData,
	WRITE_FRU_DATA_CMD:              writeFruData,

	GET_SDR_REPOSITORY_INFO_CMD:       getSdrRepositoryInfo,
	GET_SDR_REPOSITORY_ALLOC_INFO_CMD: getSdrRepositoryAllocInfo,
	RESERVE_SDR_REPOSITORY_CMD:        reserveSdrRepository,
	GET_SDR_CMD:                       getSdr,
	ADD_SDR_CMD:                       addSdr,
	PARTIAL_ADD_SDR_CMD:               partialAddSdr,
	DELETE_SDR_CMD:                    deleteSdr,
	CLEAR_SDR_REPOSITORY_CMD:          clearSdrRepository,
	GET_SDR_REPOSITORY_TIME_CMD:       getSdrRepositoryTime,
	SET_SDR_REPOSITORY_TIME_CMD:       setSdrRepositoryTime,
	ENTER_SDR_REPOSITORY_UPDATE_CMD:   enterSdrRepositoryUpdate,
	EXIT_SDR_REPOSITORY_UPDATE_CMD:    exitSdrRepositoryUpdate,
	RUN_INITIALIZATION_AGENT_CMD:      runInitializationAgent,

	GET_SEL_INFO_CMD:             getSelInfo,
	GET_SEL_ALLOCATION_INFO_CMD:  getSelAllocationInfo,
	RESERVE_SEL_CMD:              reserveSel,
	GET_SEL_ENTRY_CMD:            getSelEntry,
	ADD_SEL_ENTRY_CMD:            addSelEntry,
	PARTIAL_ADD_SEL_ENTRY_CMD:    partialAddSelEntry,
	DELETE_SEL_ENTRY_CMD:         deleteSelEntry,
	CLEAR_SEL_CMD:                clearSel,
	GET_SEL_TIME_CMD:             getSelTime,
	SET_SEL_TIME_CMD:             setSelTime,
	GET_AUXILIARY_LOG_STATUS_CMD: getAuxiliaryLogStatus,
	SET_AUXILIARY_LOG_STATUS_CMD: setAuxiliaryLogStatus,
}

func storageNetfn(msg *msgT) {
	(storageProcessors[msg.rmcp.message.cmd])(msg)
}

type transportProcessor func(*msgT)

var transportProcessors = map[uint8]transportProcessor{
	SET_LAN_CONFIG_PARMS_CMD:  setLanConfigParms,
	GET_LAN_CONFIG_PARMS_CMD:  getLanConfigParms,
	SUSPEND_BMC_ARPS_CMD:      suspendBmcArps,
	GET_IP_UDP_RMCP_STATS_CMD: getIpUdpRmcpStats,

	SET_SERIAL_MODEM_CONFIG_CMD:     setSerialModemConfig,
	GET_SERIAL_MODEM_CONFIG_CMD:     getSerialModemConfig,
	SET_SERIAL_MODEM_MUX_CMD:        setSerialModemMux,
	GET_TAP_RESPONSE_CODES_CMD:      getTapResponseCodes,
	SET_PPP_UDP_PROXY_XMIT_DATA_CMD: setPppUdpProxyXmitData,
	GET_PPP_UDP_PROXY_XMIT_DATA_CMD: getPppUdpProxyXmitData,
	SEND_PPP_UDP_PROXY_PACKET_CMD:   sendPppUdpProxyPacket,
	GET_PPP_UDP_PROXY_RECV_DATA_CMD: getPppUdpProxyRecvData,
	SERIAL_MODEM_CONN_ACTIVE_CMD:    serialModemConnActive,
	CALLBACK_CMD:                    callbackCmd,
	SET_USER_CALLBACK_OPTIONS_CMD:   setUserCallbackOptions,
	GET_USER_CALLBACK_OPTIONS_CMD:   getUserCallbackOptions,

	SOL_ACTIVATING_CMD:               solActivating,
	SET_SOL_CONFIGURATION_PARAMETERS: setSolConfigurationParameters,
	GET_SOL_CONFIGURATION_PARAMETERS: getSolConfigurationParameters,
}

func transportNetfn(msg *msgT) {
	fmt.Println("transportNetfn not supported",
		msg.rmcp.message.cmd)
}

func groupExtensionNetfn(msg *msgT) {
	fmt.Println("groupExtensionNetfn not supported",
		msg.rmcp.message.cmd)
}

func oemGroupNetfn(msg *msgT) {
	fmt.Println("oemGroupNetfn not supported",
		msg.rmcp.message.cmd)
}

const ASF_IANA = 4542

func asfPing(msg *msgT) {
	var rsp [28]uint8
	dataStart := msg.dataStart

	// Check message integrity and if it's a ping.
	if msg.dataLen < 12 {
		return
	}
	if binary.LittleEndian.Uint32(msg.data[dataStart+4:dataStart+8]) != ASF_IANA {
		return // Not ASF IANA
	}
	if msg.data[dataStart+8] != 0x80 {
		return // Not a presence ping.
	}

	// Ok, it's a valid RMCP/ASF Presence Ping
	rsp[0] = 6
	rsp[1] = 0
	rsp[2] = 0xff // No ack since it's not required, so we don't do it.
	rsp[3] = 6    // ASF class
	binary.LittleEndian.PutUint32(rsp[4:8], ASF_IANA)
	rsp[8] = 0x40                  // Presense Pong
	rsp[9] = msg.data[dataStart+9] // Message tag
	rsp[10] = 0
	rsp[11] = 16 // Data length
	// no special capabilities
	binary.LittleEndian.PutUint32(rsp[12:16], ASF_IANA)
	binary.LittleEndian.PutUint32(rsp[16:20], 0)
	rsp[20] = 0x81 // We support IPMI
	rsp[21] = 0x0  // No supported interactions
	rsp[22] = 0x0  // Reserved
	rsp[23] = 0x0  // Reserved
	rsp[24] = 0x0  // Reserved
	rsp[25] = 0x0  // Reserved
	rsp[26] = 0x0  // Reserved
	rsp[27] = 0x0  // Reserved

	if debug {
		fmt.Println("Sending ASF Ping Pong")
	}

	// Return the response.
	msg.conn.WriteToUDP(rsp[0:28], msg.remoteAddr)
}
