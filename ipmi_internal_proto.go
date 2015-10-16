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

type lanparmDataT struct {
	setInProgress   uint8
	numDestinations uint8

	ipAddrSrc        uint8
	ipAddr           net.Addr
	macAddr          [6]uint8
	subnetMask       net.Addr
	defaultGwIpAddr  net.Addr
	defaultGwMacAddr [6]uint8
	backupGwIpAddr   net.Addr
	backupGwMacAddr  [6]uint8

	vlanId                [2]uint8
	vlanPriority          uint8
	numCipherSuites       uint8
	cipherSuiteEntry      [17]uint8
	maxPrivForCipherSuite [9]uint8
}

type lanservT struct {
	lanParms              lanparmDataT
	lanAddr               net.Addr
	lanAddrSet            bool
	port                  uint16
	chanNum               uint8
	chanPrivLimit         uint8
	chanPrivAllowedAuths  [5]uint8
	activeSessions        uint8
	nextChallSeq          uint32
	sidSeq                uint32
	defaultSessionTimeout uint32
	users                 [MAX_USERS + 1]userT
	sessions              [MAX_SESSIONS + 1]sessionT
}

// For now make this global and make it more
// modular later.
var lanserv lanservT

type userT struct {
	valid        bool
	linkAuth     uint8
	cbOnly       uint8
	username     []uint8
	pw           []uint8
	privilege    uint8
	maxPriv      uint8
	maxSessions  uint8
	currSessions uint8
	allowedAuths uint16

	// Set by the user code.
	idx uint8 // My idx in the table.
}

func findUser(username []uint8, nameOnlyLookup bool, priv uint8) *userT {
	var foundUser *userT

	for i := 1; i <= MAX_USERS; i++ {
		if bytes.Equal(username, lanserv.users[i].username) {
			if nameOnlyLookup ||
				lanserv.users[i].privilege == priv {
				foundUser = &lanserv.users[i]
				break
			}
		}
	}

	if foundUser != nil {
		if debug {
			fmt.Println("findUser: ", string(foundUser.username))
		}
	}
	return foundUser
}

type sessionT struct {
	active    bool
	inStartup bool
	rmcpplus  bool

	handle uint32 // My index in the table.

	recvSeq uint32
	xmitSeq uint32
	sid     uint32
	userid  uint8

	timeLeft uint32

	/* RMCP data */
	authtype uint8
	//authdata ipmi_authdata_t

	/* RMCP+ data */
	unauthRecvSeq uint32
	unauthXmitSeq uint32
	remSid        uint32
	auth          uint8
	conf          uint8
	integ         uint
	priv          uint8
	maxPriv       uint8
}

type msgT struct {
	srcAddr interface{}
	srcLen  int

	oemData int64 /* For use by OEM handlers.  This will be set to
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
			rmcpSeq  uint8
			class    uint8
		}

		// IPMI Session layer
		session struct {
			authType    uint8
			seq         uint32
			sid         uint32
			authCode    [16]uint8
			payloadLgth uint8
		}

		// IPMI Message layer
		message struct {
			rsAddr uint8
			netfn  uint8
			rsLun  uint8
			rqAddr uint8
			rqSeq  uint8
			rqLun  uint8
			cmd    uint8
		}
	}
	// Not yet supported
	rmcpp struct {
		/* RMCP+ parms */
		payload       uint8
		encrypted     uint8
		authenticated uint8
		iana          [3]uint8
		payloadId     uint16
		authdata      *uint8
		authdataLen   uint
	}

	conn       *net.UDPConn
	remoteAddr *net.UDPAddr
	data       [4000]uint8
	dataStart  uint
	dataLen    uint

	iana uint32
}

type rspMsgDataT struct {
	netfn   uint8
	cmd     uint8
	dataLen uint16
	data    [1000]uint8
}

func ipmiChecksum(data []uint8, size int, start int8) int8 {
	csum := start

	for dataIdx := 0; size > 0; size-- {
		csum += int8(data[dataIdx])
		dataIdx++
	}

	return -csum
}

func sidToSession(sid uint32) *sessionT {
	var (
		idx     uint32
		session *sessionT
	)

	if debug {
		fmt.Printf("sidToSession: %x\n", sid)
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
