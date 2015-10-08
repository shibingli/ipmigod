// copyright 2015 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style license described in the
// LICENSE file.

// Package contains IPMI 2.0 spec protocol definitions
package ipmigod

import (
	"bytes"
	"fmt"
	"net"
)

//
// Restrictions: <=64 sessions
//
const (
	SESSION_BITS_REQ = 6 // Bits required to hold a session
	SESSION_MASK     = 0x3f
	MAX_USERS        = 64
	MAX_SESSIONS     = 16
)

var debug bool = false

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
		if debug {
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

type msg_t struct {
	src_addr interface{}
	src_len  int

	oem_data int64 /* For use by OEM handlers.  This will be set to
	   zero by the calling code. */

	channel uint8

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

func ipmi_checksum(data []uint8, size int, start int8) int8 {
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

	if debug {
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
