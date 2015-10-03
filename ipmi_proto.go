// Copyright 2015 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style license described in the
// LICENSE file.

// Package contains IPMI 2.0 spec implementation
package ipmigod

import (
	"bytes"
	"encoding/binary"
	"fmt"
	. "github.com/platinasystems/goes/cli"
	"log"
	"net"
)

const (
	MAX_USERS    = 64
	MAX_SESSIONS = 16
)

const debug = 0

type lanparm_data_t struct {
	set_in_progress  uint8
	num_destinations uint8

	ip_addr_src         uint8
	ip_addr             net.Addr
	mac_addr            [6]uint8
	subnet_mask         net.Addr
	default_gw_ip_addr  net.Addr
	default_gw_mac_addr [6]uint8
	backup_gw_ip_addr   net.Addr
	backup_gw_mac_addr  [6]uint8

	vlan_id                   [2]uint8
	vlan_priority             uint8
	num_cipher_suites         uint8
	cipher_suite_entry        [17]uint8
	max_priv_for_cipher_suite [9]uint8
}

type lanserv_t struct {
	lan_parms               lanparm_data_t
	lan_addr                net.Addr
	lan_addr_set            bool
	port                    uint16
	chan_num                uint8
	chan_priv_limit         uint8
	chan_priv_allowed_auths [5]uint8
	active_sessions         uint8
	next_chall_seq          uint32
	sid_seq                 uint32
	default_session_timeout uint32
	users                   [MAX_USERS + 1]user_t
	sessions                [MAX_SESSIONS + 1]session_t
}

// For now make this global and make it more
// modular later.
var lanserv lanserv_t

type user_t struct {
	valid         bool
	link_auth     uint8
	cb_only       uint8
	username      []uint8
	pw            []uint8
	privilege     uint8
	max_priv      uint8
	max_sessions  uint8
	curr_sessions uint8
	allowed_auths uint16

	// Set by the user code.
	idx uint8 // My idx in the table.
}

func find_user(username []uint8, name_only_lookup bool, priv uint8) *user_t {
	var found_user *user_t

	for i := 1; i <= MAX_USERS; i++ {
		if bytes.Equal(username, lanserv.users[i].username) {
			if name_only_lookup ||
				lanserv.users[i].privilege == priv {
				found_user = &lanserv.users[i]
				break
			}
		}
	}

	if found_user != nil {
		if debug > 0 {
			fmt.Println("find_user: ", string(found_user.username))
		}
	}
	return found_user
}

type session_t struct {
	active     bool
	in_startup bool
	rmcpplus   bool

	handle uint32 // My index in the table.

	recv_seq uint32
	xmit_seq uint32
	sid      uint32
	userid   uint8

	time_left uint32

	/* RMCP data */
	authtype uint8
	//authdata ipmi_authdata_t

	/* RMCP+ data */
	unauth_recv_seq uint32
	unauth_xmit_seq uint32
	rem_sid         uint32
	auth            uint8
	conf            uint8
	integ           uint

	priv     uint8
	max_priv uint8
}

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
		if debug > 0 {
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
		if debug > 0 {
			fmt.Println("Received RMCP message!")
		}
		(netfunc_processors[msg.rmcp.message.netfn])(msg)
	}
}

func (msg *msg_t) ipmi_parse_msg() {
	data_start := msg.data_start

	// Peek ahead to see if we have an RMCP or RMCP+ message
	if msg.data[data_start+4] == IPMI_AUTHTYPE_RMCP_PLUS {
		fmt.Println("LAN msg not supported RMCP+")
		//ipmi_parse_rmcpp_msg(msg)
	} else {
		msg.ipmi_parse_rmcp_msg()
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
	// hack for freeipmi - littleendian send
	//msg.rmcp.session.seq =
	//	binary.BigEndian.Uint32(msg.data[data_start+1 : data_start+5])
	//msg.rmcp.session.sid =
	//	binary.BigEndian.Uint32(msg.data[data_start+5 : data_start+9])
	msg.rmcp.session.seq =
		binary.LittleEndian.Uint32(msg.data[data_start+1 : data_start+5])
	msg.rmcp.session.sid =
		binary.LittleEndian.Uint32(msg.data[data_start+5 : data_start+9])
	if debug > 0 {
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

func ipmb_checksum(data []uint8, size int, start int8) int8 {
	csum := start

	for data_idx := 0; size > 0; size-- {
		csum += int8(data[data_idx])
		data_idx++
	}

	return -csum
}

func sid_to_session(sid uint32) *session_t {
	var (
		idx     uint32
		session *session_t
	)

	if debug > 0 {
		fmt.Printf("sid_to_session: %x\n", sid)
	}
	if sid&1 == 1 {
		return nil
	}
	idx = (sid >> 1) & SESSION_MASK
	if idx > MAX_SESSIONS {
		return nil
	}
	session = &lanserv.sessions[idx]
	if !session.active {
		return nil
	}
	if session.sid != sid {
		return nil
	}
	return session
}

func (msg *msg_t) return_rsp(session *session_t, rsp *rsp_msg_data_t) {
	var (
		data          [64]uint8
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
	// hack for freeipmi - send everything littleendian to it
	//binary.BigEndian.PutUint32(data[dcur:dcur+4], session.xmit_seq)
	binary.LittleEndian.PutUint32(data[dcur:dcur+4], session.xmit_seq)
	session.xmit_seq++
	if session.xmit_seq == 0 {
		session.xmit_seq++
	}
	dcur += 4
	// hack for freeipmi - send everything littleendian to it
	//binary.BigEndian.PutUint32(data[dcur:dcur+4], session.sid)
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
	data[dcur] = uint8(ipmb_checksum(data[start_of_msg:start_of_msg+2], 2, 0))
	if debug > 0 {
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
	csum = -ipmb_checksum(data[dcur-3:dcur], 3, 0)
	csum = ipmb_checksum(data[dcur:dcur+int(rsp.data_len)],
		int(rsp.data_len), csum)
	dcur += int(rsp.data_len)
	data[dcur] = uint8(csum)
	if debug > 0 {
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
	if debug > 0 {
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

func handle_smi_msg() {
	// send in channel and need to find emu
	//ipmi_emu_handle_msg(data->emu, chan->mc, msg, msgd, &msgd_len);
	// --> breakout to new files

	//ipmi_handle_smi_rsp(chan, msg, msgd, msgd_len);

}
