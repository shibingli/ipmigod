// Copyright 2015 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style license described in the
// LICENSE file.

// Package contains IPMI 2.0 spec implementation
package ipmigod

import (
	"encoding/binary"
	"fmt"
	_ "github.com/platinasystems/goes/cli"
	_ "log"
	"net"
	"time"
)

const (
	CLIENT_RSADDR        = 0x20
	CLIENT_RQADDR        = 0x81
	PLAT_USERNAME        = "ipmiusr"
	MAX_RETRIES          = 3
	INITIAL_OUTBOUND_SEQ = 0x3C2FB505
)

type csBuildMsg func(reqLen uint8) (data []uint8)
type csParseRsp func(data []uint8) (stateDone bool)

type clientState struct {
	buildmsg csBuildMsg
	parsersp csParseRsp
	reqLen   uint8
}

// State table to track requests to establish session
var stateTable = []clientState{
	getChannelAuthCapSt,
	getSessionChallengeSt,
	activateSessionSt,
	setSessionPrivSt,
}

var getChannelAuthCapSt = clientState{
	gcacBuildMsg,
	gcacParseRsp,
	0x09,
}

var getSessionChallengeSt = clientState{
	gscBuildMsg,
	gscParseRsp,
	0x18,
}

var activateSessionSt = clientState{
	asBuildMsg,
	asParseRsp,
	0x1D,
}

var setSessionPrivSt = clientState{
	sspBuildMsg,
	sspParseRsp,
	0x08,
}

// Structure to keep running state for this client session
// Server response variables from a message exchange are stored
// here for future message exchanges.
type clientContextT struct {
	rqSeq         uint8
	challengeStr  [16]uint8 // from getSessionChallenge
	tempSessionId uint32    // from getSessionChallenge
	sessionSeq    uint32    // from activateSession
	sessionId     uint32    // from activateSession
	maxPrivLevel  uint8     // from activateSession
	privLevel     uint8     // from setSessionPrivilege
}

var clientCtx = clientContextT{
	rqSeq: 1,
}

// Establishes an IPMI session with remote card
func ipmiEstablishSession(conn net.Conn) {

	var (
		msgData []uint8
		try     int
	)

	for idx := 0; idx < 4; idx++ {
		clState := stateTable[idx]
		// Build state-specific message
		msgData = clState.buildmsg(clState.reqLen)
		if debug {
			fmt.Printf("clientBuildMsg: % x\n", msgData[:])
		}
		for try = 0; try < MAX_RETRIES; try++ {
			if ipmiReqRsp(conn, msgData, clState.parsersp) {
				break
			}
		}
		if try >= MAX_RETRIES {
			fmt.Println("state", idx)
			panic("ipmiClient can't progress past state")
		}
	}
}

func ipmiReqRsp(conn net.Conn, msgData []uint8, parsersp csParseRsp) bool {
	var (
		stateDone bool
		rspData   [MAX_MSG_RETURN_DATA]uint8
	)

	// Send request
	_, err := conn.Write(msgData)
	if err != nil {
		fmt.Println("Error writing data to server",
			err)
		time.Sleep(500 * time.Millisecond)
		return false
	}
	// Wait for response from remote card
	if debug {
		fmt.Println("localaddr:",
			conn.LocalAddr().(*net.UDPAddr))
	}
	n, err := conn.Read(rspData[:])
	if err != nil {
		fmt.Println("Error reading data")
		fmt.Println(err)
		time.Sleep(500 * time.Millisecond)
		return false
	}
	// Parse the response and validate
	if debug {
		fmt.Printf("clientParseMsg: % x\n", rspData[:n])
	}
	stateDone = parsersp(rspData[:])
	if stateDone {
		return true
	} else {
		// pause and drop thru to retry
		time.Sleep(500 * time.Millisecond)
		return false
	}
}

// Build IPMI RMCP, Session and Message headers and Command data
func clientBuildMsg(cmdData []uint8, cmdLen uint8, reqLen uint8, sseq uint32,
	sid uint32, rsLun uint8, netFn uint8, rqLun uint8, rqSeq uint8,
	cmd uint8) []uint8 {

	var (
		data [MAX_MSG_RETURN_DATA]uint8
		csum int8
	)

	dcur := 0
	data[dcur] = 6 // RMCP version.
	dcur++
	data[dcur] = 0
	dcur++
	data[dcur] = 0xff // sequence num
	dcur++
	data[dcur] = 7 // IPMI msg class
	dcur++
	data[dcur] = 0 // authentication type
	dcur++
	binary.LittleEndian.PutUint32(data[dcur:dcur+4], sseq) // session seq
	dcur += 4
	binary.LittleEndian.PutUint32(data[dcur:dcur+4], sid) // session id
	dcur += 4

	data[dcur] = reqLen // message hdr + command-data + checksum
	dcur++
	startOfMsg := dcur
	data[dcur] = CLIENT_RSADDR
	dcur++
	data[dcur] = (netFn << 2) | rsLun
	dcur++
	data[dcur] = uint8(ipmiChecksum(data[startOfMsg:startOfMsg+2], 2, 0))
	if debug {
		fmt.Printf("csum1: %x\n", data[dcur])
	}
	dcur++
	data[dcur] = CLIENT_RQADDR
	dcur++
	data[dcur] = (rqSeq << 2) | rqLun
	dcur++
	data[dcur] = cmd
	dcur++

	// copy the cmd-specific payload data into msg data
	copy(data[dcur:], cmdData[0:cmdLen])
	csum = -ipmiChecksum(data[dcur-3:dcur], 3, 0)
	csum = ipmiChecksum(data[dcur:dcur+int(cmdLen)],
		int(cmdLen), csum)
	dcur += int(cmdLen)
	data[dcur] = uint8(csum)
	if debug {
		fmt.Printf("csum2: %x\n", data[dcur])
	}
	dcur++
	if debug {
		fmt.Println("Sending", dcur, " bytes")
	}
	return data[:uint8(dcur)]

}

func clientBasicMsgCheck(data []uint8) bool {
	if data[0] == 6 &&
		data[2] == 0xFF &&
		data[3] == 0x7 {
		// FIXME - add checksum verification here
		return true
	}
	return false
}

func gcacBuildMsg(reqLen uint8) []uint8 {
	var (
		cmdData [2]uint8
		msg     []uint8
	)
	cmdData[0] = 0x0E // channel E
	cmdData[1] = 0x03 // max priv level (operator)

	msg = clientBuildMsg(cmdData[:], uint8(len(cmdData)), reqLen, 0, 0,
		0, APP_NETFN, 0, clientCtx.rqSeq,
		GET_CHANNEL_AUTH_CAPABILITIES_CMD)
	clientCtx.rqSeq++

	return msg[:]
}

func gcacParseRsp(data []uint8) bool {

	if clientBasicMsgCheck(data) == false {
		fmt.Println("gcacParseRsp basic check failed")
		return false
	}

	var cmdOffset uint8 = 19
	if data[13] == 0x10 &&
		data[cmdOffset] == GET_CHANNEL_AUTH_CAPABILITIES_CMD &&
		data[cmdOffset+1] == 0 {
		if debug {
			fmt.Println("gcacParseRsp GOOD")
		}
		return true
	}
	return false
}

func gscBuildMsg(reqLen uint8) []uint8 {
	var (
		cmdData [17]uint8
		msg     []uint8
	)
	cmdData[0] = 0x00                                           // authtype
	copy(cmdData[1:uint8(1+len(PLAT_USERNAME))], PLAT_USERNAME) // username

	msg = clientBuildMsg(cmdData[:], uint8(len(cmdData)), reqLen, 0, 0,
		0, APP_NETFN, 0, clientCtx.rqSeq, GET_SESSION_CHALLENGE_CMD)
	clientCtx.rqSeq++

	return msg[:]
}

func gscParseRsp(data []uint8) bool {

	if clientBasicMsgCheck(data) == false {
		fmt.Println("gscParseRsp basic check failed")
		return false
	}
	var cmdOffset uint8 = 19
	if data[13] == 0x1C &&
		data[cmdOffset] == GET_SESSION_CHALLENGE_CMD &&
		data[cmdOffset+1] == 0 {
		clientCtx.tempSessionId =
			binary.LittleEndian.Uint32(data[cmdOffset+2 : cmdOffset+6])
		copy(clientCtx.challengeStr[:], data[cmdOffset+6:cmdOffset+22])
		if debug {
			fmt.Println("gscParseRsp GOOD")
		}
		return true
	}
	return false
}

func asBuildMsg(reqLen uint8) []uint8 {
	var (
		cmdData [22]uint8
		msg     []uint8
	)
	cmdData[0] = 0x00 // auth type
	cmdData[1] = 0x03 // max priv
	copy(cmdData[2:18],
		clientCtx.challengeStr[:]) // from getSessionChallenge
	binary.LittleEndian.PutUint32(cmdData[18:22], INITIAL_OUTBOUND_SEQ)
	msg = clientBuildMsg(cmdData[:], uint8(len(cmdData)), reqLen, 0,
		clientCtx.tempSessionId, 0, APP_NETFN, 0,
		clientCtx.rqSeq, ACTIVATE_SESSION_CMD)
	clientCtx.rqSeq++

	return msg[:]
}

func asParseRsp(data []uint8) bool {
	if clientBasicMsgCheck(data) == false {
		fmt.Println("asParseRsp basic check failed")
		return false
	}
	var cmdOffset uint8 = 19
	if data[13] == 0x12 &&
		data[cmdOffset] == ACTIVATE_SESSION_CMD &&
		data[cmdOffset+1] == 0 {
		clientCtx.sessionId =
			binary.LittleEndian.Uint32(data[cmdOffset+3 : cmdOffset+7])
		clientCtx.sessionSeq =
			binary.LittleEndian.Uint32(data[cmdOffset+7 : cmdOffset+11])
		clientCtx.maxPrivLevel = data[cmdOffset+11]

		if debug {
			fmt.Println("asParseRsp GOOD")
		}
		return true
	}
	return false
}

func sspBuildMsg(reqLen uint8) []uint8 {
	var (
		cmdData [1]uint8
		msg     []uint8
	)
	cmdData[0] = clientCtx.maxPrivLevel
	msg = clientBuildMsg(cmdData[:], uint8(len(cmdData)), reqLen,
		clientCtx.sessionSeq, clientCtx.sessionId, 0, APP_NETFN, 0,
		clientCtx.rqSeq, SET_SESSION_PRIVILEGE_CMD)
	clientCtx.rqSeq++
	clientCtx.sessionSeq++

	return msg[:]
}

func sspParseRsp(data []uint8) bool {
	if clientBasicMsgCheck(data) == false {
		fmt.Println("sspParseRsp basic check failed")
		return false
	}
	var cmdOffset uint8 = 19
	if data[13] == 0x09 &&
		data[cmdOffset] == SET_SESSION_PRIVILEGE_CMD &&
		data[cmdOffset+1] == 0 {
		clientCtx.privLevel = data[cmdOffset+2]
		if debug {
			fmt.Println("sspParseRsp GOOD")
		}
		return true
	}
	return false
}

func addSdrBuildMsg(sdr *sdrT) []uint8 {
	var (
		cmdData []uint8
		msg     []uint8
	)
	cmdData = append(cmdData, sdr.data[:]...)
	msg = clientBuildMsg(cmdData[:], uint8(len(cmdData)), 70,
		clientCtx.sessionSeq, clientCtx.sessionId, 0, STORAGE_NETFN, 0,
		clientCtx.rqSeq, ADD_SDR_CMD)
	clientCtx.rqSeq++
	clientCtx.sessionSeq++
	if debug {
		fmt.Printf("addSdrBuildMsg: % x\n", msg[:])
	}

	return msg[:]
}

func addSdrParseRsp(data []uint8) bool {
	if clientBasicMsgCheck(data) == false {
		fmt.Println("addSdrParseRsp basic check failed")
		return false
	}
	var cmdOffset uint8 = 19
	if data[13] == 0x0a &&
		data[cmdOffset] == ADD_SDR_CMD &&
		data[cmdOffset+1] == 0 {
		if debug {
			fmt.Println("addSdrParseRsp GOOD")
		}
		return true
	}
	return false
}
