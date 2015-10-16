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

func getDeviceId(msg *msgT) {
}

func coldReset(msg *msgT) {
}

func warmReset(msg *msgT) {
}

func getSelfTestResults(msg *msgT) {
}

func manufacturingTestOn(msg *msgT) {
}

func setAcpiPowerState(msg *msgT) {
}

func getAcpiPowerState(msg *msgT) {
}

func getDeviceGuid(msg *msgT) {
}

func resetWatchdogTimer(msg *msgT) {
}

func setWatchdogTimer(msg *msgT) {
}

func getWatchdogTimer(msg *msgT) {
}

func setBmcGlobalEnables(msg *msgT) {
}

func getBmcGlobalEnables(msg *msgT) {
}

func clearMsgFlags(msg *msgT) {
}

func getMsgFlagsCmd(msg *msgT) {
}

func enableMessageChannelRcv(msg *msgT) {
}

func getMsg(msg *msgT) {
}

func sendMsg(msg *msgT) {
}

func readEventMsgBuffer(msg *msgT) {
}

func getBtInterfaceCapabilties(msg *msgT) {
}

func getSystemGuid(msg *msgT) {
	// no session only allowed with authtype_none
	if msg.rmcp.session.sid == 0 {
		if msg.rmcp.session.authType != IPMI_AUTHTYPE_NONE {
			fmt.Println("systemGuid - no session with authtype",
				msg.rmcp.session.authType)
			return
		}
	}

}

func getChannelAuthCapabilties(msg *msgT) {
	var data [9]uint8

	if debug {
		fmt.Println("get chan auth caps message")
	}
	// no session only allowed with authtype_none
	if msg.rmcp.session.sid == 0 {

		if msg.rmcp.session.authType != IPMI_AUTHTYPE_NONE {
			fmt.Println("systemGuid - no session with authtype",
				msg.rmcp.session.authType)
			return
		}
	}

	dataStart := msg.dataStart
	do_rmcpp := (msg.data[dataStart] >> 7) & 1
	if do_rmcpp > 0 {
		fmt.Println("get chan auth caps: rmcpp requested")
		return
	}

	channel := msg.data[dataStart] & 0xf
	priv := msg.data[msg.dataStart+1] & 0xf
	if channel == 0xe { // means use "this channel"
		channel = lanserv.chanNum
	}
	if channel != lanserv.chanNum {
		fmt.Println("get chan auth caps: chan mismatch ", channel,
			lanserv.chanNum)
		msg.returnErr(nil, IPMI_INVALID_DATA_FIELD_CC)
	} else if priv > lanserv.chanPrivLimit {
		fmt.Println("get chan auth caps: priv problem ", priv,
			lanserv.chanNum)
		msg.returnErr(nil, IPMI_INVALID_DATA_FIELD_CC)
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
		msg.returnRspData(nil, data[0:9], 9)
	}

}

func isAuthvalNull(authval []uint8) bool {
	for i := 0; i < 16; i++ {
		if authval[i] != 0 {
			return false
		}
	}
	return true
}

func getSessionChallenge(msg *msgT) {
	var (
		user *userT
		data [21]uint8
		sid  uint32
	)

	// no-session only allowed with authtype_none
	if msg.rmcp.session.sid == 0 {

		if msg.rmcp.session.authType != IPMI_AUTHTYPE_NONE {
			fmt.Println("systemGuid - no session with authtype",
				msg.rmcp.session.authType)
			return
		}
	}

	dataStart := msg.dataStart
	authtype := msg.data[dataStart] & 0xf
	user = findUser(msg.data[dataStart+1:dataStart+17], true, authtype)
	if user == nil {
		if isAuthvalNull(msg.data[dataStart+1 : dataStart+17]) {
			msg.returnErr(nil, 0x82) // no null user
		} else {
			msg.returnErr(nil, 0x81) // no user
		}
		return
	}

	if (user.allowedAuths & (1 << authtype)) == 0 {
		fmt.Println("Session challenge failed: Invalid auth type",
			authtype)
		msg.returnErr(nil, IPMI_INVALID_DATA_FIELD_CC)
		return
	}

	if lanserv.activeSessions >= MAX_SESSIONS {
		fmt.Println("Session challenge failed: Too many open sessions")
		msg.returnErr(nil, IPMI_OUT_OF_SPACE_CC)
		return
	}

	data[0] = 0

	sid = (lanserv.nextChallSeq << (USER_BITS_REQ + 1)) |
		(uint32(user.idx) << 1) | 1
	lanserv.nextChallSeq++
	if debug {
		fmt.Printf("Temp-session-id: %x\n", sid)
	}
	binary.LittleEndian.PutUint32(data[1:5], sid)
	if debug {
		fmt.Printf("Temp-session-id: %x\n", data[1:5])
	}

	//rv = gen_challenge(lan, data+5, sid)
	testChall := []uint8{0x00, 0x01, 0x02, 0x03,
		0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b,
		0x0c, 0x0d, 0x0e, 0x0f}
	copy(data[5:], testChall[0:])
	msg.returnRspData(nil, data[0:21], 21)
}

func findFreeSession() *sessionT {

	// Find a free session. Session 0 is invalid.
	for i := 1; i <= MAX_SESSIONS; i++ {
		if !lanserv.sessions[i].active {
			return &lanserv.sessions[i]
		}
	}
	return nil
}

// TODO: authcode support
func activateSession(msg *msgT) {
	var (
		data    [11]uint8
		session *sessionT
	)

	if msg.dataLen < 22 {
		fmt.Println("Activate session fail: message too short")
		return
	}

	// Handle temporary session case (i.e. no session established yet)
	if msg.rmcp.session.sid&1 == 1 {

		var dummySession sessionT
		dataStart := msg.dataStart

		// check_challenge() - not for now!

		// establish new session under lan struct and calc new sid
		userIdx := (msg.sid >> 1) & USER_MASK
		if (userIdx > MAX_USERS) || (userIdx == 0) {
			fmt.Println("Activate session invalid sid",
				msg.sid)
			return
		}

		auth := msg.data[dataStart] & 0xf
		user := &(lanserv.users[userIdx])
		if !user.valid {
			fmt.Println("Activate session invalid ui ", userIdx)
			return
		}
		if (user.allowedAuths & (1 << auth)) == 0 {
			fmt.Println("Activate session invalid auth for user ",
				auth, userIdx)
			return
		}
		if (user.allowedAuths & (1 << msg.authtype)) == 0 {
			fmt.Println("Activate session invalid msg auth user ",
				msg.authtype, userIdx)
			return
		}

		if lanserv.activeSessions >= MAX_SESSIONS {
			fmt.Println("Activate session fail:  Too many open!")
			return
		}

		xmitSeq :=
			binary.LittleEndian.Uint32(msg.data[dataStart+18 : dataStart+22])

		dummySession.active = true
		dummySession.authtype = msg.authtype
		dummySession.xmitSeq = xmitSeq
		dummySession.sid = msg.sid

		if xmitSeq == 0 {
			fmt.Println("Activate session fail:  xmitSeq 0")
			msg.returnErr(&dummySession, 0x85) // Invalid xmitseq
			return
		}

		maxPriv := msg.data[dataStart+1] & 0xf
		if (user.privilege == 0xf) || (maxPriv > user.maxPriv) {
			fmt.Println("Activate session fail: priv mismatch",
				maxPriv, user.maxPriv)
			msg.returnErr(&dummySession, 0x86) //Priv err
			return
		}

		session = findFreeSession()
		if session == nil {
			fmt.Println("Activate session fail: no free sessions")
			msg.returnErr(&dummySession, 0x81) // No session slot
			return
		}

		session.active = true
		session.rmcpplus = false
		session.authtype = auth

		r := rand.New(rand.NewSource(99))
		seqData := r.Uint32()
		session.recvSeq = seqData & 0xFFFFFFFE
		if session.recvSeq == 0 {
			session.recvSeq = 2
		}
		session.xmitSeq = xmitSeq
		session.maxPriv = maxPriv
		session.priv = IPMI_PRIVILEGE_USER // Start at user privilege
		session.userid = user.idx
		session.timeLeft = lanserv.defaultSessionTimeout

		lanserv.activeSessions++
		if debug {
			fmt.Printf("Activate session: Session opened\n")
			fmt.Printf("0x%x, max priv %d\n", userIdx, maxPriv)
		}

		if lanserv.sidSeq == 0 {
			lanserv.sidSeq++
		}
		session.sid =
			uint32((lanserv.sidSeq << (SESSION_BITS_REQ + 1)) |
				(session.handle << 1))
		lanserv.sidSeq++

		// Build response and send back
		data[0] = 0
		data[1] = session.authtype

		binary.LittleEndian.PutUint32(data[2:6], session.sid)
		binary.LittleEndian.PutUint32(data[6:10], session.recvSeq)

		data[10] = session.maxPriv

		msg.returnRspData(&dummySession, data[0:11], 11)

	} else {
		// actiavate_session msg while already in a session
		session = sidToSession(msg.sid)
		if session == nil {
			fmt.Printf("Activate session - no session %x\n",
				msg.sid)
			return
		}

		// We are already connected, we ignore everything
		// but the outbound sequence number.
		session.xmitSeq = binary.LittleEndian.Uint32(msg.data[18:22])

		// Build response and send back
		data[0] = 0
		data[1] = session.authtype

		binary.LittleEndian.PutUint32(data[2:6], session.sid)
		binary.LittleEndian.PutUint32(data[6:10], session.recvSeq)

		data[10] = session.maxPriv

		msg.returnRspData(session, data[0:11], 11)

	}
	fmt.Printf("\nSession %d activated\n", session.handle)
}

func setSessionPrivilege(msg *msgT) {
	var (
		data [2]uint8
		priv uint8
	)

	if msg.dataLen < 1 {
		fmt.Printf("Set session priv msg too short %d\n", msg.dataLen)
		msg.returnErr(nil, IPMI_REQUEST_DATA_LENGTH_INVALID_CC)
		return
	}

	session := sidToSession(msg.sid)
	if session == nil {
		fmt.Printf("Set session priv - no session %x\n", msg.sid)
		return
	}

	priv = msg.data[msg.dataStart] & 0xf

	if priv == 0 {
		priv = session.priv
	}

	if priv == IPMI_PRIVILEGE_CALLBACK {
		fmt.Println("Set session priv - can't drop below user priv")
		msg.returnErr(session, 0x80) // Can't drop below user priv
		return
	}

	if priv > session.maxPriv {
		msg.returnErr(session, 0x81) // Can't set priv this high
		return
	}

	session.priv = priv

	data[0] = 0
	data[1] = priv

	msg.returnRspData(session, data[0:2], 2)
}

func closeSession(msg *msgT) {

	session := sidToSession(msg.sid)
	var sid uint32
	targetSess := session

	if msg.dataLen < 4 {
		fmt.Printf("Close session failure: message too short %d\n",
			msg.dataLen)
		msg.returnErr(session, IPMI_REQUEST_DATA_LENGTH_INVALID_CC)
		return
	}

	dataStart := msg.dataStart
	sid = binary.LittleEndian.Uint32(msg.data[dataStart : dataStart+4])
	if sid != session.sid {
		// Close session from another session
		if session.priv != IPMI_PRIVILEGE_ADMIN {
			// Only admins can close other people's sessions.
			fmt.Println("Session mismatch on close session")
			msg.returnErr(session,
				IPMI_INSUFFICIENT_PRIVILEGE_CC)
			return
		}
		targetSess = sidToSession(sid)
		if targetSess == nil {
			msg.returnErr(session, 0x87) /* session not found */
			return
		}
	}

	fmt.Printf("Session %d closed: Closed due to request\n",
		targetSess.handle)

	// Send ack on originating session
	msg.returnErr(session, 0)

	// Cleanup the target session (which could be the same or not)
	targetSess.active = false
	lanserv.activeSessions--

}

func getSessionInfo(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func getAuthcode(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func setChannelAccess(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func getChannelAccess(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func getChannelInfo(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func setUserAccess(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func getUserAccess(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func setUserName(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func getUserName(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func setUserPassword(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func activatePayload(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func deavtivatePayload(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func getPayloadActivationStatus(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func getPayloadInstanceInfo(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func setUserPayloadAccess(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func getUserPayloadAccess(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func getChannelPayloadSupport(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func getChannelPayloadVersion(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func getChannelOemPayloadInfo(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func masterReadWrite(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func getChannelCipherSuites(msg *msgT) {
	// no session only allowed with authtype_none
	if msg.rmcp.session.sid == 0 {

		if msg.rmcp.session.authType != IPMI_AUTHTYPE_NONE {
			fmt.Println("system_guid - no session with authtype",
				msg.rmcp.session.authType)
			return
		}
	}

}

func suspendResumePayloadEncryption(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func setChannelSecurityKey(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}

func getSystemInterfaceCapabilities(msg *msgT) {
	fmt.Println("appNetfn not supported", msg.rmcp.message.cmd)
}
