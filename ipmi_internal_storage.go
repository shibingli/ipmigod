// Copyright 2015 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style license described in the
// LICENSE file.

// Package contains IPMI 2.0 spec protocol definitions
package ipmigod

import (
	"encoding/binary"
	"fmt"
)

const (
	MAX_SDR_LENGTH = 261
	MAX_NUM_SDRS   = 1024
)

func getFruInventoryAreaInfo(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func readFruData(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func writeFruData(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func getSdrRepositoryInfo(msg *msgT) {
	var data [15]uint8

	data[0] = 0
	data[1] = 0x51
	binary.LittleEndian.PutUint16(data[2:4], mc.mainSdrs.sdrCount)
	space := MAX_SDR_LENGTH * (MAX_NUM_SDRS - mc.mainSdrs.sdrCount)
	if space > 0xfffe {
		space = 0xfffe
	}
	binary.LittleEndian.PutUint16(data[4:6], space)
	binary.LittleEndian.PutUint32(data[6:10],
		mc.mainSdrs.lastAddTime)
	binary.LittleEndian.PutUint32(data[10:14],
		mc.mainSdrs.lastEraseTime)
	data[14] = mc.mainSdrs.flags

	msg.returnRspData(nil, data[0:15], 15)
}

func getSdrRepositoryAllocInfo(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func reserveSdrRepository(msg *msgT) {
	var data [3]uint8

	mc.mainSdrs.reservation++
	if mc.mainSdrs.reservation == 0 {
		mc.mainSdrs.reservation++
	}

	data[0] = 0
	binary.LittleEndian.PutUint16(data[1:3], mc.mainSdrs.reservation)

	msg.returnRspData(nil, data[0:3], 3)
}

func getSdr(msg *msgT) {

	var (
		data  [MAX_MSG_RETURN_DATA]uint8
		entry *sdrT
	)

	dataStart := msg.dataStart
	reservation :=
		binary.LittleEndian.Uint16(msg.data[dataStart : dataStart+2])

	if reservation != 0 && reservation != mc.mainSdrs.reservation {
		fmt.Println("getSdr: reservation mismatch", reservation,
			mc.mainSdrs.reservation)
		msg.returnErr(nil, IPMI_INVALID_RESERVATION_CC)
		return
	}

	recordId :=
		binary.LittleEndian.Uint16(msg.data[dataStart+2 : dataStart+4])
	offset := msg.data[dataStart+4]
	count := msg.data[dataStart+5]

	if recordId == 0 {
		entry = mc.mainSdrs.sdrs
	} else if recordId == 0xffff {
		entry = mc.mainSdrs.tailSdr
	} else {
		entry = mc.mainSdrs.sdrs
		for entry != nil {
			if entry.recordId == recordId {
				break
			}
			entry = entry.next
		}
	}

	if entry == nil {
		fmt.Println("getSdr: Can't find recordId", recordId)
		msg.returnErr(nil, IPMI_NOT_PRESENT_CC)
		return
	}

	if offset >= entry.length {
		fmt.Println("getSdr: offset out of range")
		msg.returnErr(nil, IPMI_PARAMETER_OUT_OF_RANGE_CC)
		return
	}

	if (offset + count) > entry.length {
		count = entry.length - offset
	}
	if uint(count+3) > MAX_MSG_RETURN_DATA {
		fmt.Println("getSdr: cannot return required data")
		// Too much data to put into response.
		msg.returnErr(nil, IPMI_CANNOT_RETURN_REQ_LENGTH_CC)
		return
	}

	data[0] = 0
	if entry.next != nil {
		binary.LittleEndian.PutUint16(data[1:3],
			entry.next.recordId)
	} else {
		data[1] = 0xff
		data[2] = 0xff
	}

	copy(data[3:], entry.data[offset:offset+count])
	msg.returnRspData(nil, data[0:], uint(count+3))
}

func addSdrCmd(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func partialAddSdr(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func deleteSdr(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func clearSdrRepository(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func getSdrRepositoryTime(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func setSdrRepositoryTime(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func enterSdrRepositoryUpdate(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func exitSdrRepositoryUpdate(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func runInitializationAgent(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func getSelInfo(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func getSelAllocationInfo(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func reserveSel(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func getSelEntry(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func addSelEntry(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func partialAddSelEntry(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func deleteSelEntry(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func clearSel(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func getSelTime(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func setSelTime(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func getAuxiliaryLogStatus(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func setAuxiliaryLogStatus(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}
