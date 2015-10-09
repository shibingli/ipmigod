// Copyright 2015 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style license described in the
// LICENSE file.

// Package contains IPMI 2.0 spec protocol definitions
package ipmigod

import (
	"encoding/binary"
	"fmt"
	"math/rand"
)

const (
	USER_BITS_REQ = 6
	USER_MASK     = 0x3f
)

func get_device_id(msg *msg_t) {
}

func cold_reset(msg *msg_t) {
}

func warm_reset(msg *msg_t) {
}

func get_self_test_results(msg *msg_t) {
}

func manufacturing_test_on(msg *msg_t) {
}

func set_acpi_power_state(msg *msg_t) {
}

func get_acpi_power_state(msg *msg_t) {
}

func get_device_guid(msg *msg_t) {
}

func reset_watchdog_timer(msg *msg_t) {
}

func set_watchdog_timer(msg *msg_t) {
}

func get_watchdog_timer(msg *msg_t) {
}

func set_bmc_global_enables(msg *msg_t) {
}

func get_bmc_global_enables(msg *msg_t) {
}

func clear_msg_flags(msg *msg_t) {
}

func get_msg_flags_cmd(msg *msg_t) {
}

func enable_message_channel_rcv(msg *msg_t) {
}

func get_msg(msg *msg_t) {
}

func send_msg(msg *msg_t) {
}

func read_event_msg_buffer(msg *msg_t) {
}

func get_bt_interface_capabilties(msg *msg_t) {
}

func get_system_guid(msg *msg_t) {
	// no session only allowed with authtype_none
	if msg.rmcp.session.sid == 0 {
		if msg.rmcp.session.auth_type != IPMI_AUTHTYPE_NONE {
			fmt.Println("system_guid - no session with authtype",
				msg.rmcp.session.auth_type)
			return
		}
	}

}

func get_channel_auth_capabilties(msg *msg_t) {
	var data [9]uint8

	if debug {
		fmt.Println("get chan auth caps message")
	}
	// no session only allowed with authtype_none
	if msg.rmcp.session.sid == 0 {

		if msg.rmcp.session.auth_type != IPMI_AUTHTYPE_NONE {
			fmt.Println("system_guid - no session with authtype",
				msg.rmcp.session.auth_type)
			return
		}
	}

	data_start := msg.data_start
	do_rmcpp := (msg.data[data_start] >> 7) & 1
	if do_rmcpp > 0 {
		fmt.Println("get chan auth caps: rmcpp requested")
		return
	}

	channel := msg.data[data_start] & 0xf
	priv := msg.data[msg.data_start+1] & 0xf
	if channel == 0xe { // means use "this channel"
		channel = lanserv.chan_num
	}
	if channel != lanserv.chan_num {
		fmt.Println("get chan auth caps: chan mismatch ", channel,
			lanserv.chan_num)
		msg.return_err(nil, IPMI_INVALID_DATA_FIELD_CC)
	} else if priv > lanserv.chan_priv_limit {
		fmt.Println("get chan auth caps: priv problem ", priv,
			lanserv.chan_num)
		msg.return_err(nil, IPMI_INVALID_DATA_FIELD_CC)
	} else {
		data[0] = 0
		data[1] = channel
		data[2] = 0x17 //HACK lanserv.chan_priv_allowed_auths[priv-1]
		data[3] = 0x6  // HACK per-message authentication is on,
		// Only RMCP for now (no RMCPP)
		// user-level authenitcation is on,
		// non-null user names disabled,
		// no anonymous support.
		data[4] = 0
		data[5] = 0
		data[6] = 0
		data[7] = 0
		data[8] = 0
		msg.return_rsp_data(nil, data[0:9], 9)
	}

}

func is_authval_null(authval []uint8) bool {
	for i := 0; i < 16; i++ {
		if authval[i] != 0 {
			return false
		}
	}
	return true
}

func get_session_challenge(msg *msg_t) {
	var (
		user *user_t
		data [21]uint8
		sid  uint32
	)

	// no-session only allowed with authtype_none
	if msg.rmcp.session.sid == 0 {

		if msg.rmcp.session.auth_type != IPMI_AUTHTYPE_NONE {
			fmt.Println("system_guid - no session with authtype",
				msg.rmcp.session.auth_type)
			return
		}
	}

	data_start := msg.data_start
	authtype := msg.data[data_start] & 0xf
	user = find_user(msg.data[data_start+1:data_start+17], true, authtype)
	if user == nil {
		if is_authval_null(msg.data[data_start+1 : data_start+17]) {
			msg.return_err(nil, 0x82) // no null user
		} else {
			msg.return_err(nil, 0x81) // no user
		}
		return
	}

	if (user.allowed_auths & (1 << authtype)) == 0 {
		fmt.Println("Session challenge failed: Invalid auth type",
			authtype)
		msg.return_err(nil, IPMI_INVALID_DATA_FIELD_CC)
		return
	}

	if lanserv.active_sessions >= MAX_SESSIONS {
		fmt.Println("Session challenge failed: Too many open sessions")
		msg.return_err(nil, IPMI_OUT_OF_SPACE_CC)
		return
	}

	data[0] = 0

	sid = (lanserv.next_chall_seq << (USER_BITS_REQ + 1)) |
		(uint32(user.idx) << 1) | 1
	lanserv.next_chall_seq++
	if debug {
		fmt.Printf("Temp-session-id: %x\n", sid)
	}
	binary.LittleEndian.PutUint32(data[1:5], sid)
	if debug {
		fmt.Printf("Temp-session-id: %x\n", data[1:5])
	}

	//rv = gen_challenge(lan, data+5, sid)
	test_chall := []uint8{0x00, 0x01, 0x02, 0x03,
		0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b,
		0x0c, 0x0d, 0x0e, 0x0f}
	copy(data[5:], test_chall[0:])
	msg.return_rsp_data(nil, data[0:21], 21)
}

func find_free_session() *session_t {

	// Find a free session. Session 0 is invalid.
	for i := 1; i <= MAX_SESSIONS; i++ {
		if !lanserv.sessions[i].active {
			return &lanserv.sessions[i]
		}
	}
	return nil
}

// TODO: authcode support
func activate_session(msg *msg_t) {
	var (
		data    [11]uint8
		session *session_t
	)

	if msg.data_len < 22 {
		fmt.Println("Activate session fail: message too short")
		return
	}

	// Handle temporary session case (i.e. no session established yet)
	if msg.rmcp.session.sid&1 == 1 {

		var dummy_session session_t
		data_start := msg.data_start

		// check_challenge() - not for now!

		// establish new session under lan struct and calc new sid
		user_idx := (msg.sid >> 1) & USER_MASK
		if (user_idx > MAX_USERS) || (user_idx == 0) {
			fmt.Println("Activate session invalid sid",
				msg.sid)
			return
		}

		auth := msg.data[data_start] & 0xf
		user := &(lanserv.users[user_idx])
		if !user.valid {
			fmt.Println("Activate session invalid ui ", user_idx)
			return
		}
		if (user.allowed_auths & (1 << auth)) == 0 {
			fmt.Println("Activate session invalid auth for user ",
				auth, user_idx)
			return
		}
		if (user.allowed_auths & (1 << msg.authtype)) == 0 {
			fmt.Println("Activate session invalid msg auth user ",
				msg.authtype, user_idx)
			return
		}

		if lanserv.active_sessions >= MAX_SESSIONS {
			fmt.Println("Activate session fail:  Too many open!")
			return
		}

		xmit_seq :=
			binary.LittleEndian.Uint32(msg.data[data_start+18 : data_start+22])

		dummy_session.active = true
		dummy_session.authtype = msg.authtype
		dummy_session.xmit_seq = xmit_seq
		dummy_session.sid = msg.sid

		if xmit_seq == 0 {
			fmt.Println("Activate session fail:  xmit_seq 0")
			msg.return_err(&dummy_session, 0x85) // Invalid xmitseq
			return
		}

		max_priv := msg.data[data_start+1] & 0xf
		if (user.privilege == 0xf) || (max_priv > user.max_priv) {
			fmt.Println("Activate session fail: priv mismatch",
				max_priv, user.max_priv)
			msg.return_err(&dummy_session, 0x86) //Priv err
			return
		}

		session = find_free_session()
		if session == nil {
			fmt.Println("Activate session fail: no free sessions")
			msg.return_err(&dummy_session, 0x81) // No session slot
			return
		}

		session.active = true
		session.rmcpplus = false
		session.authtype = auth

		r := rand.New(rand.NewSource(99))
		seq_data := r.Uint32()
		session.recv_seq = seq_data & 0xFFFFFFFE
		if session.recv_seq == 0 {
			session.recv_seq = 2
		}
		session.xmit_seq = xmit_seq
		session.max_priv = max_priv
		session.priv = IPMI_PRIVILEGE_USER // Start at user privilege
		session.userid = user.idx
		session.time_left = lanserv.default_session_timeout

		lanserv.active_sessions++
		if debug {
			fmt.Printf("Activate session: Session opened\n")
			fmt.Printf("0x%x, max priv %d\n", user_idx, max_priv)
		}

		if lanserv.sid_seq == 0 {
			lanserv.sid_seq++
		}
		session.sid =
			uint32((lanserv.sid_seq << (SESSION_BITS_REQ + 1)) |
				(session.handle << 1))
		lanserv.sid_seq++

		// Build response and send back
		data[0] = 0
		data[1] = session.authtype

		binary.LittleEndian.PutUint32(data[2:6], session.sid)
		binary.LittleEndian.PutUint32(data[6:10], session.recv_seq)

		data[10] = session.max_priv

		msg.return_rsp_data(&dummy_session, data[0:11], 11)

	} else {
		// actiavate_session msg while already in a session
		session = sid_to_session(msg.sid)
		if session == nil {
			fmt.Printf("Activate session - no session %x\n",
				msg.sid)
			return
		}

		// We are already connected, we ignore everything
		// but the outbound sequence number.
		session.xmit_seq = binary.LittleEndian.Uint32(msg.data[18:22])

		// Build response and send back
		data[0] = 0
		data[1] = session.authtype

		binary.LittleEndian.PutUint32(data[2:6], session.sid)
		binary.LittleEndian.PutUint32(data[6:10], session.recv_seq)

		data[10] = session.max_priv

		msg.return_rsp_data(session, data[0:11], 11)

	}
	fmt.Printf("\nSession %d activated\n", session.handle)
}

func set_session_privilege(msg *msg_t) {
	var (
		data [2]uint8
		priv uint8
	)

	if msg.data_len < 1 {
		fmt.Printf("Set session priv msg too short %d\n", msg.data_len)
		msg.return_err(nil, IPMI_REQUEST_DATA_LENGTH_INVALID_CC)
		return
	}

	session := sid_to_session(msg.sid)
	if session == nil {
		fmt.Printf("Set session priv - no session %x\n", msg.sid)
		return
	}

	priv = msg.data[msg.data_start] & 0xf

	if priv == 0 {
		priv = session.priv
	}

	if priv == IPMI_PRIVILEGE_CALLBACK {
		fmt.Println("Set session priv - can't drop below user priv")
		msg.return_err(session, 0x80) // Can't drop below user priv
		return
	}

	if priv > session.max_priv {
		msg.return_err(session, 0x81) // Can't set priv this high
		return
	}

	session.priv = priv

	data[0] = 0
	data[1] = priv

	msg.return_rsp_data(session, data[0:2], 2)
}

func close_session(msg *msg_t) {

	session := sid_to_session(msg.sid)
	var sid uint32
	target_sess := session

	if msg.data_len < 4 {
		fmt.Printf("Close session failure: message too short %d\n",
			msg.data_len)
		msg.return_err(session, IPMI_REQUEST_DATA_LENGTH_INVALID_CC)
		return
	}

	data_start := msg.data_start
	sid = binary.LittleEndian.Uint32(msg.data[data_start : data_start+4])
	if sid != session.sid {
		// Close session from another session
		if session.priv != IPMI_PRIVILEGE_ADMIN {
			// Only admins can close other people's sessions.
			fmt.Println("Session mismatch on close session")
			msg.return_err(session,
				IPMI_INSUFFICIENT_PRIVILEGE_CC)
			return
		}
		target_sess = sid_to_session(sid)
		if target_sess == nil {
			msg.return_err(session, 0x87) /* session not found */
			return
		}
	}

	fmt.Printf("Session %d closed: Closed due to request\n",
		target_sess.handle)

	// Send ack on originating session
	msg.return_err(session, 0)

	// Cleanup the target session (which could be the same or not)
	target_sess.active = false
	lanserv.active_sessions--

}

func get_session_info(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func get_authcode(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func set_channel_access(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func get_channel_access(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func get_channel_info(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func set_user_access(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func get_user_access(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func set_user_name(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func get_user_name(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func set_user_password(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func activate_payload(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func deavtivate_payload(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func get_payload_activation_status(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func get_payload_instance_info(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func set_user_payload_access(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func get_user_payload_access(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func get_channel_payload_support(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func get_channel_payload_version(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func get_channel_oem_payload_info(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func master_read_write(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func get_channel_cipher_suites(msg *msg_t) {
	// no session only allowed with authtype_none
	if msg.rmcp.session.sid == 0 {

		if msg.rmcp.session.auth_type != IPMI_AUTHTYPE_NONE {
			fmt.Println("system_guid - no session with authtype",
				msg.rmcp.session.auth_type)
			return
		}
	}

}

func suspend_resume_payload_encryption(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func set_channel_security_key(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}

func get_system_interface_capabilities(msg *msg_t) {
	fmt.Println("app_netfn not supported", msg.rmcp.message.cmd)
}
