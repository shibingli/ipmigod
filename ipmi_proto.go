// Copyright 2015 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style license described in the
// LICENSE file.

// Package contains IPMI 2.0 spec implementation
package ipmigod

import (
	"encoding/binary"
	"fmt"
	. "github.com/platinasystems/goes/cli"
	"log"
	"net"
)

const (
	MAX_MSG_RETURN_DATA = 1000
)

func ipmi_lan_init() {

	// Initialize user database for straight authentication
	lanserv.users[1].idx = 1
	lanserv.users[1].username = make([]uint8, 16)
	copy(lanserv.users[1].username[0:], "")
	lanserv.users[1].pw = make([]uint8, 16)
	copy(lanserv.users[1].pw[0:], "test")
	lanserv.users[1].max_priv = IPMI_PRIVILEGE_USER
	lanserv.users[1].allowed_auths = (1 << IPMI_AUTHTYPE_NONE) |
		//(1 << AUTHTYPE_MD2) |
		//(1 << AUTHTYPE_MD5) |
		(1 << IPMI_AUTHTYPE_STRAIGHT)
	lanserv.users[1].valid = true
	lanserv.users[2].idx = 2
	lanserv.users[2].username = make([]uint8, 16)
	copy(lanserv.users[2].username[0:], "ipmiusr")
	lanserv.users[2].pw = make([]uint8, 16)
	copy(lanserv.users[2].pw[0:], "test")
	lanserv.users[2].max_priv = IPMI_PRIVILEGE_ADMIN
	lanserv.users[2].allowed_auths = (1 << IPMI_AUTHTYPE_NONE) //|
	//(1 << AUTHTYPE_MD2) |
	//(1 << AUTHTYPE_MD5) |
	//(1 << IPMI_AUTHTYPE_STRAIGHT)
	lanserv.users[2].valid = true

	lanserv.chan_num = 1
	lanserv.default_session_timeout = 30
	lanserv.sid_seq = 0
	lanserv.next_chall_seq = 0
	lanserv.chan_priv_limit = IPMI_PRIVILEGE_ADMIN
	lanserv.chan_priv_allowed_auths[IPMI_PRIVILEGE_CALLBACK-1] =
		(1 << IPMI_AUTHTYPE_MD5)
	lanserv.chan_priv_allowed_auths[IPMI_PRIVILEGE_USER-1] =
		(1 << IPMI_AUTHTYPE_NONE)
	lanserv.chan_priv_allowed_auths[IPMI_PRIVILEGE_OPERATOR-1] =
		(1 << IPMI_AUTHTYPE_NONE)
	lanserv.chan_priv_allowed_auths[IPMI_PRIVILEGE_ADMIN-1] =
		(1 << IPMI_AUTHTYPE_STRAIGHT)
	lanserv.chan_priv_allowed_auths[IPMI_PRIVILEGE_OEM-1] =
		(1 << IPMI_AUTHTYPE_OEM)

	for i := 1; i < MAX_SESSIONS+1; i++ {
		lanserv.sessions[i].handle = uint32(i)
	}

}

func init() {
	// goes setup
	const name = "ipmigod"
	Apropos.Set(name, `ipmigod daemon`)

	//Complete.Set(name, complete)
	//Help.Set(name, help)
	Usage.Set(name, `ipmigod [OPTIONS]...`)
	Command.Set(name, func(_ *Context, _ ...string) {
		ipmigod_main()
	})

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
	ipmi_lan_init()

	// Initialize BMC SDRs/Sensors
	bmc_init()

	// Initialize persistence database
}

func ipmigod_main() {

	// Listen on UDP port 623 on all interfaces.
	server_addr, err := net.ResolveUDPAddr("udp", ":623")
	if err != nil {
		log.Fatal(err)
	}

	// Now listen at selected port
	server_conn, err := net.ListenUDP("udp", server_addr)
	if err != nil {
		log.Fatal(err)
	}
	defer server_conn.Close()

	for {

		msg := new(msg_t)
		n, remote_addr, err :=
			server_conn.ReadFromUDP(msg.data[0:])
		msg.remote_addr = remote_addr
		if debug {
			fmt.Println("Received ", n, " bytes from ",
				msg.remote_addr)
		}
		if err != nil {
			fmt.Println("Error: ", err)
			fmt.Printf("Error: Received %d bytes\n", n)
		}
		msg.data_len = uint(n)
		msg.conn = server_conn
		msg.ipmi_handle_msg()
	}
}

func (msg *msg_t) ipmi_handle_msg() {

	if msg.data_len < 5 {
		fmt.Printf("LAN msg failure: message too short %d",
			msg.data_len)
		return
	}
	msg.channel = lanserv.chan_num

	// Parse incoming IPMI packet (including error checks)
	// and load up msg struct
	msg.ipmi_parse_msg()

	if msg.authtype == IPMI_AUTHTYPE_RMCP_PLUS {
		//ipmi_handle_rmcpp_msg(lan, &msg);
		fmt.Println("Received RMCP+ message!")
	} else {
		if debug {
			fmt.Println("Received RMCP message!")
		}
		(netfunc_processors[msg.rmcp.message.netfn])(msg)
	}
}

func (msg *msg_t) ipmi_parse_msg() {
	data_start := msg.data_start

	if msg.data[data_start+3] == 6 {
		// Handle ASF ping message
		asf_ping(msg)
	} else if msg.data[data_start+3] == 7 {
		// Peek ahead to see if we have an RMCP or RMCP+ message
		if msg.data[data_start+4] == IPMI_AUTHTYPE_RMCP_PLUS {
			fmt.Println("LAN msg not supported RMCP+")
			//ipmi_parse_rmcpp_msg(msg)
		} else {
			msg.ipmi_parse_rmcp_msg()
		}
	} else {
		fmt.Println("LAN msg has unsupported class",
			msg.data[data_start+3])
	}
}

func (msg *msg_t) ipmi_parse_rmcp_msg() {
	data_start := msg.data_start

	// Load RMCP header
	msg.rmcp.hdr.version = msg.data[data_start+0]
	msg.rmcp.hdr.rmcp_seq = msg.data[data_start+2]
	msg.rmcp.hdr.class = msg.data[data_start+3]
	msg.data_start += 4
	data_start = msg.data_start

	if msg.rmcp.hdr.rmcp_seq != 0xff {
		fmt.Println("LAN msg failure: seq not ff")
		return /* Sequence # must be ff (no ack) */
	}

	// Load IPMI Session fields
	msg.rmcp.session.auth_type = msg.data[data_start]
	msg.authtype = msg.rmcp.session.auth_type
	msg.rmcp.session.seq =
		binary.LittleEndian.Uint32(msg.data[data_start+1 : data_start+5])
	msg.rmcp.session.sid =
		binary.LittleEndian.Uint32(msg.data[data_start+5 : data_start+9])
	if debug {
		fmt.Printf("Session_id from freeipmi: %x\n",
			msg.rmcp.session.sid)
	}
	msg.sid = msg.rmcp.session.sid
	if msg.rmcp.session.auth_type != IPMI_AUTHTYPE_NONE {
		copy(msg.rmcp.session.auth_code[0:],
			msg.data[data_start+9:data_start+25])
		msg.rmcp.session.payload_lgth = msg.data[data_start+25]
		msg.data_start += 26
	} else {
		msg.data_start += 10
	}
	data_start = msg.data_start

	// Load IPMI Message fields
	msg.rmcp.message.rs_addr = msg.data[data_start]
	msg.rmcp.message.netfn = msg.data[data_start+1] >> 2
	msg.rmcp.message.rs_lun = msg.data[data_start+1] & 0x3
	msg.rmcp.message.rq_addr = msg.data[data_start+3]
	msg.rmcp.message.rq_seq = msg.data[data_start+4] >> 2
	msg.rmcp.message.rq_lun = msg.data[data_start+4] & 0x3
	msg.rmcp.message.cmd = msg.data[data_start+5]
	msg.data_start += 6
}

func (msg *msg_t) return_rsp(session *session_t, rsp *rsp_msg_data_t) {
	var (
		data          [MAX_MSG_RETURN_DATA]uint8
		csum          int8
		dummy_session session_t
		len           int
	)

	if session == nil {
		session = sid_to_session(msg.sid)
	}
	if session != nil && session.rmcpplus {
		//rmcp plus not currently supported
		fmt.Println("RMCP return_rsp not supported!")
		return
	} else if msg.sid == 0 {
		session = &dummy_session
		session.active = true
		session.authtype = IPMI_AUTHTYPE_NONE
		session.xmit_seq = 0
		session.sid = 0
	}

	if session == nil {
		fmt.Println("return_rsp: Can't find session")
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
	binary.LittleEndian.PutUint32(data[dcur:dcur+4], session.xmit_seq)
	session.xmit_seq++
	if session.xmit_seq == 0 {
		session.xmit_seq++
	}
	dcur += 4
	binary.LittleEndian.PutUint32(data[dcur:dcur+4], session.sid)
	dcur += 4
	if session.authtype != IPMI_AUTHTYPE_NONE {
		dcur += 16 // sizeof rmcp.session.auth_code[]
	}
	// Add message structure length to specified payload length
	len = int(rsp.data_len + 7) // rmcp.message layer size
	data[dcur] = uint8(len)
	dcur++
	start_of_msg := dcur
	data[dcur] = msg.rmcp.message.rq_addr
	dcur++
	data[dcur] = (rsp.netfn << 2) | msg.rmcp.message.rq_lun
	dcur++
	data[dcur] = uint8(ipmi_checksum(data[start_of_msg:start_of_msg+2], 2, 0))
	if debug {
		fmt.Printf("csum1: %x\n", data[dcur])
	}
	dcur++
	data[dcur] = msg.rmcp.message.rs_addr
	dcur++
	data[dcur] = (msg.rmcp.message.rq_seq << 2) | msg.rmcp.message.rs_lun
	dcur++
	data[dcur] = rsp.cmd
	dcur++
	// copy the response payload data into msg data
	copy(data[dcur:], rsp.data[0:rsp.data_len])
	csum = -ipmi_checksum(data[dcur-3:dcur], 3, 0)
	csum = ipmi_checksum(data[dcur:dcur+int(rsp.data_len)],
		int(rsp.data_len), csum)
	dcur += int(rsp.data_len)
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
		fmt.Println("Sending", dcur, " bytes to", msg.remote_addr)
	}
	msg.conn.WriteToUDP(data[0:dcur], msg.remote_addr)
}

func (msg *msg_t) return_err(session *session_t, err uint8) {

	var rsp rsp_msg_data_t

	rsp.netfn = msg.rmcp.message.netfn | 1
	rsp.cmd = msg.rmcp.message.cmd
	rsp.data[0] = err
	rsp.data_len = 1
	msg.return_rsp(session, &rsp)
}

func (msg *msg_t) return_rsp_data(session *session_t, data []uint8,
	data_length uint) {
	var rsp rsp_msg_data_t

	rsp.netfn = msg.rmcp.message.netfn | 1
	rsp.cmd = msg.rmcp.message.cmd
	copy(rsp.data[0:], data[0:data_length])
	rsp.data_len = uint16(data_length)

	msg.return_rsp(session, &rsp)
}

type ipmi_netfunc_processor func(*msg_t)

var netfunc_processors = map[uint8]ipmi_netfunc_processor{
	CHASSIS_NETFN:         chassis_netfn,
	BRIDGE_NETFN:          bridge_netfn,
	SENSOR_EVENT_NETFN:    sensor_event_netfn,
	APP_NETFN:             app_netfn,
	FIRMWARE_NETFN:        firmware_netfn,
	STORAGE_NETFN:         storage_netfn,
	TRANSPORT_NETFN:       transport_netfn,
	GROUP_EXTENSION_NETFN: group_extension_netfn,
	OEM_GROUP_NETFN:       oem_group_netfn,
}

type chassis_processor func(*msg_t)

var chassis_processors = map[uint8]chassis_processor{
	GET_CHASSIS_CAPABILITIES_CMD: get_chassis_capabilities,
	CHASSIS_CONTROL_CMD:          chassis_control,
	CHASSIS_RESET_CMD:            chassis_reset,
	CHASSIS_IDENTIFY_CMD:         chassis_identify,
	SET_CHASSIS_CAPABILITIES_CMD: set_chassis_capabilities,
	SET_POWER_RESTORE_POLICY_CMD: set_power_restore_policy,
	GET_SYSTEM_RESTART_CAUSE_CMD: get_system_restart_cause,
	SET_SYSTEM_BOOT_OPTIONS_CMD:  set_system_boot_options,
	GET_SYSTEM_BOOT_OPTIONS_CMD:  get_system_boot_options,
}

func chassis_netfn(msg *msg_t) {
	(chassis_processors[msg.rmcp.message.cmd])(msg)
}

type bridge_processor func(*msg_t)

var bridge_processors = map[uint8]bridge_processor{
	GET_BRIDGE_STATE_CMD:         get_bridge_state,
	SET_BRIDGE_STATE_CMD:         set_bridge_state,
	GET_ICMB_ADDRESS_CMD:         get_icmb_address,
	SET_ICMB_ADDRESS_CMD:         set_icmb_address,
	SET_BRIDGE_PROXY_ADDRESS_CMD: set_bridge_proxy_address,
	GET_BRIDGE_STATISTICS_CMD:    get_bridge_statistics,
	GET_ICMB_CAPABILITIES_CMD:    get_icmb_capabilities,

	CLEAR_BRIDGE_STATISTICS_CMD:  clear_bridge_statistics,
	GET_BRIDGE_PROXY_ADDRESS_CMD: get_bridge_proxy_address,
	GET_ICMB_CONNECTOR_INFO_CMD:  get_icmb_connector_info,
	SET_ICMB_CONNECTOR_INFO_CMD:  set_icmb_connector_info,
	SEND_ICMB_CONNECTION_ID_CMD:  send_icmb_connection_id,

	PREPARE_FOR_DISCOVERY_CMD: prepare_for_discovery,
	GET_ADDRESSES_CMD:         get_addresses,
	SET_DISCOVERED_CMD:        set_discovered,
	GET_CHASSIS_DEVICE_ID_CMD: get_chassis_device_id,
	SET_CHASSIS_DEVICE_ID_CMD: set_chassis_device_id,

	BRIDGE_REQUEST_CMD: bridge_request,
	BRIDGE_MESSAGE_CMD: bridge_message,

	GET_EVENT_COUNT_CMD:           get_event_count,
	SET_EVENT_DESTINATION_CMD:     set_event_destination,
	SET_EVENT_RECEPTION_STATE_CMD: set_event_reception_state,
	SEND_ICMB_EVENT_MESSAGE_CMD:   send_icmb_event_message,
	GET_EVENT_DESTIATION_CMD:      get_event_destination,
	GET_EVENT_RECEPTION_STATE_CMD: get_event_reception_state,

	ERROR_REPORT_CMD: error_report,
}

func bridge_netfn(msg *msg_t) {
	fmt.Println("bridge_netfn not supported",
		msg.rmcp.message.cmd)
}

type sensor_processor func(*msg_t)

var sensor_processors = map[uint8]sensor_processor{
	SET_EVENT_RECEIVER_CMD: set_event_receiver,
	GET_EVENT_RECEIVER_CMD: get_event_receiver,
	PLATFORM_EVENT_CMD:     platform_event,

	GET_PEF_CAPABILITIES_CMD:        get_pef_capabilities,
	ARM_PEF_POSTPONE_TIMER_CMD:      arm_pef_postpone_timer,
	SET_PEF_CONFIG_PARMS_CMD:        set_pef_config_parms,
	GET_PEF_CONFIG_PARMS_CMD:        get_pef_config_parms,
	SET_LAST_PROCESSED_EVENT_ID_CMD: set_last_processed_event_id,
	GET_LAST_PROCESSED_EVENT_ID_CMD: get_last_processed_event_id,
	ALERT_IMMEDIATE_CMD:             alert_immediate,
	PET_ACKNOWLEDGE_CMD:             pet_acknowledge,

	GET_DEVICE_SDR_INFO_CMD:           get_device_sdr_info,
	GET_DEVICE_SDR_CMD:                get_device_sdr,
	RESERVE_DEVICE_SDR_REPOSITORY_CMD: reserve_device_sdr_repository,
	GET_SENSOR_READING_FACTORS_CMD:    get_sensor_reading_factors,
	SET_SENSOR_HYSTERESIS_CMD:         set_sensor_hysteresis,
	GET_SENSOR_HYSTERESIS_CMD:         get_sensor_hysteresis,
	SET_SENSOR_THRESHOLD_CMD:          set_sensor_threshold,
	GET_SENSOR_THRESHOLD_CMD:          get_sensor_threshold,
	SET_SENSOR_EVENT_ENABLE_CMD:       set_sensor_event_enable,
	GET_SENSOR_EVENT_ENABLE_CMD:       get_sensor_event_enable,
	REARM_SENSOR_EVENTS_CMD:           rearm_sensor_events,
	GET_SENSOR_EVENT_STATUS_CMD:       get_sensor_event_status,
	GET_SENSOR_READING_CMD:            get_sensor_reading,
	SET_SENSOR_TYPE_CMD:               set_sensor_type,
	GET_SENSOR_TYPE_CMD:               get_sensor_type,
}

func sensor_event_netfn(msg *msg_t) {
	(sensor_processors[msg.rmcp.message.cmd])(msg)
}

type app_processor func(*msg_t)

var app_processors = map[uint8]app_processor{
	GET_DEVICE_ID_CMD:                 get_device_id,
	COLD_RESET_CMD:                    cold_reset,
	WARM_RESET_CMD:                    warm_reset,
	GET_SELF_TEST_RESULTS_CMD:         get_self_test_results,
	MANUFACTURING_TEST_ON_CMD:         manufacturing_test_on,
	SET_ACPI_POWER_STATE_CMD:          set_acpi_power_state,
	GET_ACPI_POWER_STATE_CMD:          get_acpi_power_state,
	GET_DEVICE_GUID_CMD:               get_device_guid,
	RESET_WATCHDOG_TIMER_CMD:          reset_watchdog_timer,
	SET_WATCHDOG_TIMER_CMD:            set_watchdog_timer,
	GET_WATCHDOG_TIMER_CMD:            get_watchdog_timer,
	SET_BMC_GLOBAL_ENABLES_CMD:        set_bmc_global_enables,
	GET_BMC_GLOBAL_ENABLES_CMD:        get_bmc_global_enables,
	CLEAR_MSG_FLAGS_CMD:               clear_msg_flags,
	GET_MSG_FLAGS_CMD:                 get_msg_flags_cmd,
	ENABLE_MESSAGE_CHANNEL_RCV_CMD:    enable_message_channel_rcv,
	GET_MSG_CMD:                       get_msg,
	SEND_MSG_CMD:                      send_msg,
	READ_EVENT_MSG_BUFFER_CMD:         read_event_msg_buffer,
	GET_BT_INTERFACE_CAPABILITIES_CMD: get_bt_interface_capabilties,
	GET_SYSTEM_GUID_CMD:               get_system_guid,
	GET_CHANNEL_AUTH_CAPABILITIES_CMD: get_channel_auth_capabilties,
	GET_SESSION_CHALLENGE_CMD:         get_session_challenge,
	ACTIVATE_SESSION_CMD:              activate_session,
	SET_SESSION_PRIVILEGE_CMD:         set_session_privilege,
	CLOSE_SESSION_CMD:                 close_session,
	GET_SESSION_INFO_CMD:              get_session_info,

	GET_AUTHCODE_CMD:                  get_authcode,
	SET_CHANNEL_ACCESS_CMD:            set_channel_access,
	GET_CHANNEL_ACCESS_CMD:            get_channel_access,
	GET_CHANNEL_INFO_CMD:              get_channel_info,
	SET_USER_ACCESS_CMD:               set_user_access,
	GET_USER_ACCESS_CMD:               get_user_access,
	SET_USER_NAME_CMD:                 set_user_name,
	GET_USER_NAME_CMD:                 get_user_name,
	SET_USER_PASSWORD_CMD:             set_user_password,
	ACTIVATE_PAYLOAD_CMD:              activate_payload,
	DEACTIVATE_PAYLOAD_CMD:            deavtivate_payload,
	GET_PAYLOAD_ACTIVATION_STATUS_CMD: get_payload_activation_status,
	GET_PAYLOAD_INSTANCE_INFO_CMD:     get_payload_instance_info,
	SET_USER_PAYLOAD_ACCESS_CMD:       set_user_payload_access,
	GET_USER_PAYLOAD_ACCESS_CMD:       get_user_payload_access,
	GET_CHANNEL_PAYLOAD_SUPPORT_CMD:   get_channel_payload_support,
	GET_CHANNEL_PAYLOAD_VERSION_CMD:   get_channel_payload_version,
	GET_CHANNEL_OEM_PAYLOAD_INFO_CMD:  get_channel_oem_payload_info,

	MASTER_READ_WRITE_CMD: master_read_write,

	GET_CHANNEL_CIPHER_SUITES_CMD:         get_channel_cipher_suites,
	SUSPEND_RESUME_PAYLOAD_ENCRYPTION_CMD: suspend_resume_payload_encryption,
	SET_CHANNEL_SECURITY_KEY_CMD:          set_channel_security_key,
	GET_SYSTEM_INTERFACE_CAPABILITIES_CMD: get_system_interface_capabilities,
}

func app_netfn(msg *msg_t) {
	(app_processors[msg.rmcp.message.cmd])(msg)
}

func firmware_netfn(msg *msg_t) {
	fmt.Println("firmware_netfn not supported",
		msg.rmcp.message.cmd)
}

type storage_processor func(*msg_t)

var storage_processors = map[uint8]storage_processor{
	GET_FRU_INVENTORY_AREA_INFO_CMD: get_fru_inventory_area_info,
	READ_FRU_DATA_CMD:               read_fru_data,
	WRITE_FRU_DATA_CMD:              write_fru_data,

	GET_SDR_REPOSITORY_INFO_CMD:       get_sdr_repository_info,
	GET_SDR_REPOSITORY_ALLOC_INFO_CMD: get_sdr_repository_alloc_info,
	RESERVE_SDR_REPOSITORY_CMD:        reserve_sdr_repository,
	GET_SDR_CMD:                       get_sdr,
	ADD_SDR_CMD:                       add_sdr_cmd,
	PARTIAL_ADD_SDR_CMD:               partial_add_sdr,
	DELETE_SDR_CMD:                    delete_sdr,
	CLEAR_SDR_REPOSITORY_CMD:          clear_sdr_repository,
	GET_SDR_REPOSITORY_TIME_CMD:       get_sdr_repository_time,
	SET_SDR_REPOSITORY_TIME_CMD:       set_sdr_repository_time,
	ENTER_SDR_REPOSITORY_UPDATE_CMD:   enter_sdr_repository_update,
	EXIT_SDR_REPOSITORY_UPDATE_CMD:    exit_sdr_repository_update,
	RUN_INITIALIZATION_AGENT_CMD:      run_initialization_agent,

	GET_SEL_INFO_CMD:             get_sel_info,
	GET_SEL_ALLOCATION_INFO_CMD:  get_sel_allocation_info,
	RESERVE_SEL_CMD:              reserve_sel,
	GET_SEL_ENTRY_CMD:            get_sel_entry,
	ADD_SEL_ENTRY_CMD:            add_sel_entry,
	PARTIAL_ADD_SEL_ENTRY_CMD:    partial_add_sel_entry,
	DELETE_SEL_ENTRY_CMD:         delete_sel_entry,
	CLEAR_SEL_CMD:                clear_sel,
	GET_SEL_TIME_CMD:             get_sel_time,
	SET_SEL_TIME_CMD:             set_sel_time,
	GET_AUXILIARY_LOG_STATUS_CMD: get_auxiliary_log_status,
	SET_AUXILIARY_LOG_STATUS_CMD: set_auxiliary_log_status,
}

func storage_netfn(msg *msg_t) {
	(storage_processors[msg.rmcp.message.cmd])(msg)
}

type transport_processor func(*msg_t)

var transport_processors = map[uint8]transport_processor{
	SET_LAN_CONFIG_PARMS_CMD:  set_lan_config_parms,
	GET_LAN_CONFIG_PARMS_CMD:  get_lan_config_parms,
	SUSPEND_BMC_ARPS_CMD:      suspend_bmc_arps,
	GET_IP_UDP_RMCP_STATS_CMD: get_ip_udp_rmcp_stats,

	SET_SERIAL_MODEM_CONFIG_CMD:     set_serial_modem_config,
	GET_SERIAL_MODEM_CONFIG_CMD:     get_serial_modem_config,
	SET_SERIAL_MODEM_MUX_CMD:        set_serial_modem_mux,
	GET_TAP_RESPONSE_CODES_CMD:      get_tap_response_codes,
	SET_PPP_UDP_PROXY_XMIT_DATA_CMD: set_ppp_udp_proxy_xmit_data,
	GET_PPP_UDP_PROXY_XMIT_DATA_CMD: get_ppp_udp_proxy_xmit_data,
	SEND_PPP_UDP_PROXY_PACKET_CMD:   send_ppp_udp_proxy_packet,
	GET_PPP_UDP_PROXY_RECV_DATA_CMD: get_ppp_udp_proxy_recv_data,
	SERIAL_MODEM_CONN_ACTIVE_CMD:    serial_modem_conn_active,
	CALLBACK_CMD:                    callback_cmd,
	SET_USER_CALLBACK_OPTIONS_CMD:   set_user_callback_options,
	GET_USER_CALLBACK_OPTIONS_CMD:   get_user_callback_options,

	SOL_ACTIVATING_CMD:               sol_activating,
	SET_SOL_CONFIGURATION_PARAMETERS: set_sol_configuration_parameters,
	GET_SOL_CONFIGURATION_PARAMETERS: get_sol_configuration_parameters,
}

func transport_netfn(msg *msg_t) {
	fmt.Println("transport_netfn not supported",
		msg.rmcp.message.cmd)
}

func group_extension_netfn(msg *msg_t) {
	fmt.Println("group_extension_netfn not supported",
		msg.rmcp.message.cmd)
}

func oem_group_netfn(msg *msg_t) {
	fmt.Println("oem_group_netfn not supported",
		msg.rmcp.message.cmd)
}

const ASF_IANA = 4542

func asf_ping(msg *msg_t) {
	var rsp [28]uint8
	data_start := msg.data_start

	// Check message integrity and if it's a ping.
	if msg.data_len < 12 {
		return
	}
	if binary.LittleEndian.Uint32(msg.data[data_start+4:data_start+8]) != ASF_IANA {
		return // Not ASF IANA
	}
	if msg.data[data_start+8] != 0x80 {
		return // Not a presence ping.
	}

	// Ok, it's a valid RMCP/ASF Presence Ping
	rsp[0] = 6
	rsp[1] = 0
	rsp[2] = 0xff // No ack since it's not required, so we don't do it.
	rsp[3] = 6    // ASF class
	binary.LittleEndian.PutUint32(rsp[4:8], ASF_IANA)
	rsp[8] = 0x40                   // Presense Pong
	rsp[9] = msg.data[data_start+9] // Message tag
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
	msg.conn.WriteToUDP(rsp[0:28], msg.remote_addr)
}
