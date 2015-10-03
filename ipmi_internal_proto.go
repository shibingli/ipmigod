// copyright 2015 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style license described in the
// LICENSE file.

// Package contains IPMI 2.0 spec protocol definitions
package ipmigod

import (
	"fmt"
	"net"
)

//
// Restrictions: <=64 sessions
//
const (
	SESSION_BITS_REQ = 6 // Bits required to hold a session
	SESSION_MASK     = 0x3f
)

type msg_t struct {
	src_addr interface{}
	src_len  int

	oem_data int64 /* For use by OEM handlers.  This will be set to
	   zero by the calling code. */

	channel uint8

	// The channel the message originally came in on.
	//channel_t *orig_channel;

	// shorthand fields
	sid      uint32
	authtype uint8

	rmcp struct {
		// RMCP layer
		hdr struct {
			version  uint8
			reserved uint8
			rmcp_seq uint8
			class    uint8
		}

		// IPMI Session layer
		session struct {
			auth_type    uint8
			seq          uint32
			sid          uint32
			auth_code    [16]uint8
			payload_lgth uint8
		}

		// IPMI Message layer
		message struct {
			rs_addr uint8
			netfn   uint8
			rs_lun  uint8
			rq_addr uint8
			rq_seq  uint8
			rq_lun  uint8
			cmd     uint8
		}
	}
	// Not yet supported
	rmcpp struct {
		/* RMCP+ parms */
		payload       uint8
		encrypted     uint8
		authenticated uint8
		iana          [3]uint8
		payload_id    uint16
		authdata      *uint8
		authdata_len  uint
	}

	conn        *net.UDPConn
	remote_addr *net.UDPAddr
	data        [4000]uint8
	data_start  uint
	data_len    uint

	iana uint32
}

type rsp_msg_data_t struct {
	netfn    uint8
	cmd      uint8
	data_len uint16
	data     [1000]uint8
}

type auth_data_t struct {
	rand         [16]byte
	rem_rand     [16]byte
	role         byte
	username_len byte
	username     [16]byte
	sik          [20]byte
	k1           [20]byte
	k2           [20]byte
	akey_len     byte
	integ_len    uint
	adata        interface{}
	akey         interface{}
	ikey_len     uint
	idata        interface{}
	ikey         interface{}
	ikey2        interface{}
	ckey_len     uint
	cdata        interface{}
	ckey         interface{}
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
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
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
	WRITE_FRU_DATA_CMD:              write_fur_data,

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
	fmt.Println("storage_netfn not supported",
		msg.rmcp.message.cmd)
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
