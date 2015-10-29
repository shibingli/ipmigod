// Copyright 2015 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style license described in the
// LICENSE file.

// Package contains IPMI 2.0 spec protocol definitions
package ipmigod

import (
	"encoding/binary"
	"fmt"
	"time"
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

	if debug {
		fmt.Println("getSdr: ", recordId, offset, entry.length)
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

func newSdrEntry(length uint8) *sdrT {

	newSdr := new(sdrT)
	if newSdr == nil {
		return nil
	}
	newSdr.recordId = mc.mainSdrs.nextFreeEntryId
	mc.mainSdrs.nextFreeEntryId++
	newSdr.length = length + 5

	// Have to update record data to reflect local MM state
	if chassisCardNum == 0 {
		binary.LittleEndian.PutUint16(newSdr.data[:2], newSdr.recordId)
	}

	return newSdr
}

func addSdrEntry(newSdr *sdrT) {
	if mc.mainSdrs.sdrs == nil {
		mc.mainSdrs.sdrs = newSdr
	} else {
		mc.mainSdrs.tailSdr.next = newSdr
	}
	if debug {
		fmt.Println("addSdrEntry: ", newSdr.recordId)
	}
	mc.mainSdrs.tailSdr = newSdr
	now := time.Now()
	nowUnix := uint32(now.Unix())
	mc.sel.lastAddTime = nowUnix
	mc.mainSdrs.sdrCount++
}

func addSdr(msg *msgT) {
	var (
		data  [3]uint8
		entry *sdrT
	)

	// Points directly into full SDR record data
	dataStart := msg.dataStart

	// If oem field is 0 we have a regular addSdr otherwise
	// it's really a sensor value update
	if msg.data[dataStart+46] == 0 {
		if debug {
			fmt.Printf("Received addSdr: % x\n",
				msg.data[0:msg.dataLen])
		}

		// FXME - add a check for duplicate SDRs

		entry = newSdrEntry(msg.data[dataStart+4])
		if entry == nil {
			msg.returnErr(nil, IPMI_OUT_OF_SPACE_CC)
			return
		}
		// Update Sensor number from msg
		entry.sensNum = msg.data[dataStart+7]
		copy(entry.data[2:2+entry.length],
			msg.data[dataStart+2:dataStart+2+uint(entry.length)])
		entry.enabled = true
		entry.eventsEnabled = true
		entry.scanningEnabled = true
		entry.eventStatus = 0

		addSdrEntry(entry)

		data[0] = 0
		binary.LittleEndian.PutUint16(data[1:3], entry.recordId)
		msg.returnRspData(nil, data[0:3], 3)
	} else {
		if debug {
			fmt.Printf("Received special addSdr: % x\n",
				msg.data[0:msg.dataLen])
		}

		// Find sdr entry in local SDR database and update its value
		lun := msg.data[dataStart+6]
		sensNum := msg.data[dataStart+7]
		value := msg.data[dataStart+46]
		entry = mc.mainSdrs.sdrs
		for entry != nil {
			if entry.lun == lun &&
				entry.sensNum == sensNum {
				entry.value = value
				break
			}
			entry = entry.next
		}

		if entry != nil {
			data[0] = 0
			binary.LittleEndian.PutUint16(data[1:3],
				entry.recordId)
			msg.returnRspData(nil, data[0:3], 3)
		}
	}
}

func partialAddSdr(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func deleteSdr(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func clearSdrRepository(msg *msgT) {
	var entry, n_entry *sdrT

	var data [2]uint8
	dataStart := msg.dataStart
	reservation :=
		binary.LittleEndian.Uint16(msg.data[dataStart : dataStart+2])
	if (reservation != 0) && (reservation != mc.mainSdrs.reservation) {
		msg.returnErr(nil, IPMI_INVALID_RESERVATION_CC)
		return
	}

	if (msg.data[dataStart+2] != 'C') ||
		(msg.data[dataStart+3] != 'L') ||
		(msg.data[dataStart+4] != 'R') {
		msg.returnErr(nil, IPMI_INVALID_DATA_FIELD_CC)
		return
	}

	op := msg.data[dataStart+5]
	if op != 0 && op != 0xaa {
		msg.returnErr(nil, IPMI_INVALID_DATA_FIELD_CC)
		return
	}

	data[0] = 0
	data[1] = 1
	if op == 0xaa {
		entry = mc.mainSdrs.sdrs
		for entry != nil {
			n_entry = entry.next
			// free implicit - garbage collector should free
			entry = n_entry
		}
		mc.mainSdrs.sdrs = nil
		mc.mainSdrs.tailSdr = nil
		now := time.Now()
		nowUnix := uint32(now.Unix())
		mc.mainSdrs.lastEraseTime = nowUnix
	}

	msg.returnRspData(nil, data[0:2], 2)
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
	var data [15]uint8

	data[1] = 0x51
	binary.LittleEndian.PutUint16(data[2:4], mc.sel.count)
	binary.LittleEndian.PutUint16(data[4:6],
		(mc.sel.maxCount-mc.sel.count)*16)
	binary.LittleEndian.PutUint32(data[6:10], mc.sel.lastAddTime)
	binary.LittleEndian.PutUint32(data[10:14], mc.sel.lastEraseTime)
	data[14] = mc.sel.flags

	msg.returnRspData(nil, data[0:15], 15)
}

func getSelAllocationInfo(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func reserveSel(msg *msgT) {

	var data [3]uint8

	mc.sel.reservation++
	if mc.sel.reservation == 0 {
		mc.sel.reservation++
	}

	data[0] = 0
	binary.LittleEndian.PutUint16(data[1:3], mc.sel.reservation)

	msg.returnRspData(nil, data[0:3], 3)
}

func getSelEntry(msg *msgT) {
	var (
		nextRecordId uint16
		entry        selEntryT
		data         [19]uint8
	)

	dataStart := msg.dataStart
	reservation :=
		binary.LittleEndian.Uint16(msg.data[dataStart : dataStart+2])
	if (reservation != 0) && (reservation != mc.sel.reservation) {
		msg.returnErr(nil, IPMI_INVALID_RESERVATION_CC)
		return
	}

	recordId :=
		binary.LittleEndian.Uint16(msg.data[dataStart+2 : dataStart+4])
	offset := msg.data[dataStart+4]
	count := msg.data[dataStart+5]

	if offset >= 16 {
		msg.returnErr(nil, IPMI_INVALID_DATA_FIELD_CC)
		return
	}

	if mc.sel.count == 0 {
		msg.returnErr(nil, IPMI_NOT_PRESENT_CC)
		return
	}

	// record id of 0 means 1st one in sel; 0xFF means last.
	if recordId == 0 {
		entry = mc.sel.entries[0] // 0th entry is valid
		if mc.sel.count-1 > 0 {
			nextRecordId = 2
		} else {
			nextRecordId = 0xffff
		}
	} else if recordId == 0xffff {
		entry = mc.sel.entries[mc.sel.count-1]
		nextRecordId = 0xffff
	} else {
		for i := range mc.sel.entries {
			if mc.sel.entries[i].recordId == recordId {
				entry = mc.sel.entries[i]
				if i+1 >= int(mc.sel.count) {
					nextRecordId = 0xffff
				} else {
					nextRecordId = uint16(i + 2)
				}
				break
			}
		}
	}

	// Nothing found ?
	if entry.recordId == 0 {
		msg.returnErr(nil, IPMI_NOT_PRESENT_CC)
		return
	}

	data[0] = 0
	binary.LittleEndian.PutUint16(data[1:3], nextRecordId)

	if (offset + count) > 16 {
		count = 16 - offset
	}
	copy(data[3:], entry.data[offset:offset+count])
	retLen := count + 3
	msg.returnRspData(nil, data[0:retLen], uint(retLen))

}

func addSelEntry(msg *msgT) {
	var data [19]uint8

	dataStart := msg.dataStart
	err, r := addToSel(msg.data[dataStart+2],
		msg.data[dataStart+3:dataStart+16])
	if err != 0 {
		msg.returnErr(nil, uint8(err))
		return
	} else {
		data[0] = 0
		binary.LittleEndian.PutUint16(data[1:3], r)
	}
	msg.returnRspData(nil, data[0:3], 3)
}

func partialAddSelEntry(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func deleteSelEntry(msg *msgT) {
	fmt.Println("storageNetfn not supported", msg.rmcp.message.cmd)
}

func clearSel(msg *msgT) {

	var data [2]uint8
	dataStart := msg.dataStart
	reservation :=
		binary.LittleEndian.Uint16(msg.data[dataStart : dataStart+2])
	if (reservation != 0) && (reservation != mc.sel.reservation) {
		msg.returnErr(nil, IPMI_INVALID_RESERVATION_CC)
		return
	}

	if (msg.data[dataStart+2] != 'C') ||
		(msg.data[dataStart+3] != 'L') ||
		(msg.data[dataStart+4] != 'R') {
		msg.returnErr(nil, IPMI_INVALID_DATA_FIELD_CC)
		return
	}

	op := msg.data[dataStart+5]
	if op != 0 && op != 0xaa {
		msg.returnErr(nil, IPMI_INVALID_DATA_FIELD_CC)
		return
	}

	data[0] = 0
	data[1] = 1
	if op == 0xaa {
		mc.sel.entries = nil
		// nil'ing causes capacity to go to 0 so remake
		mc.sel.entries = make([]selEntryT, mc.sel.maxCount)

		now := time.Now()
		nowUnix := uint32(now.Unix())
		mc.sel.lastEraseTime = nowUnix
		// Clear the overflow flag.
		mc.sel.flags &^= 0x80
	}

	msg.returnRspData(nil, data[0:2], 2)
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
