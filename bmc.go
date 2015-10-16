// Copyright 2015 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style license described in the
// LICENSE file.

// Package contains IPMI 2.0 spec protocol definitions
package ipmigod

import (
	"encoding/binary"
	"fmt"
)

// Used for artificial qemu environment
// Should be false when running on real hw
var simulate bool = true

// Device_support bits
const (
	IPMI_DEVID_CHASSIS_DEVICE     = (1 << 7)
	IPMI_DEVID_BRIDGE             = (1 << 6)
	IPMI_DEVID_IPMB_EVENT_GEN     = (1 << 5)
	IPMI_DEVID_IPMB_EVENT_RCV     = (1 << 4)
	IPMI_DEVID_FRU_INVENTORY_DEV  = (1 << 3)
	IPMI_DEVID_SEL_DEVICE         = (1 << 2)
	IPMI_DEVID_SDR_REPOSITORY_DEV = (1 << 1)
	IPMI_DEVID_SENSOR_DEV         = (1 << 0)
)

// Main sdr flags
const (
	IPMI_SDR_DELETE_SDR_SUPPORTED             = (1 << 3)
	IPMI_SDR_PARTIAL_ADD_SDR_SUPPORTED        = (1 << 2)
	IPMI_SDR_RESERVE_SDR_SUPPORTED            = (1 << 1)
	IPMI_SDR_GET_SDR_ALLOC_INFO_SDR_SUPPORTED = (1 << 0)
)

type sdrT struct {
	recordId uint16
	length   uint8
	data     [76]uint8
	next     *sdrT
}

type sdrs_t struct {
	reservation     uint16
	sdrCount        uint16
	sensorCount     uint16
	lastAddTime     uint32
	lastEraseTime   uint32
	timeOffset      uint64
	flags           uint8
	nextFreeEntryId uint16
	sdrsLength      uint32
	tailSdr         *sdrT
	sdrs            *sdrT // A linked list of SDR entries.

}

type mcT struct {
	bmcIpmb        uint8 // address of bmc
	deviceId       uint8
	hasDeviceSdrs  bool
	deviceRevision uint8
	majorFwRev     uint8
	minorFwRev     uint8
	deviceSupport  uint8
	mfgId          [3]uint8
	productId      [2]uint8
	mainSdrs       sdrs_t
	sensors        [4][255]*sensorT
}

var mc mcT

func bmcInit() {
	// Initialize the bmc
	mc.bmcIpmb = 0x20
	mc.deviceId = 0
	mc.hasDeviceSdrs = false
	mc.deviceRevision = 1
	mc.majorFwRev = 1
	mc.minorFwRev = 1
	mc.deviceSupport = IPMI_DEVID_SDR_REPOSITORY_DEV |
		IPMI_DEVID_SENSOR_DEV
	mc.mfgId[0] = 0
	mc.mfgId[1] = 0
	mc.mfgId[2] = 1
	mc.productId[0] = 0
	mc.productId[1] = 0

	mc.mainSdrs.flags = IPMI_SDR_RESERVE_SDR_SUPPORTED

	// Initially this is a simulated set of sensors.
	// In production, a similar scheme could be used or
	// perhaps a more dynamic scheme where the sysclass fs is
	// scanned to gather these params.
	if simulate {
		// sensor 1 - temp
		sensorAdd(0x20, 0, 1, 1, 1)
		sensorName1 := []uint8{'D', 'J', 't', 'e', 'm', 'p'}
		mainSdrAdd(0x20, 0x0001, 0x51, 1, 0x31, 0x20, 0, 1,
			3, 1, 0x67, 0x88, 1, 1, 0xC00F,
			0xC07F, 0x3838, 0, 1, 0, 0,
			1, 0, 0, 0, 0, 0, 3, 0x60,
			0xB0, 0, 0xB0, 0, 0xA0, 0x90, 0x66, 0,
			0, 0, 0, 0, 0, 0, 0, 0xC6,
			sensorName1)

		// sensor 2 - voltage
		sensorAdd(0x20, 0, 2, 2, 1)
		sensorName2 := []uint8{'M', 'X', 'v', 'o', 'l', 't', 'a', 'g', 'e'}
		mainSdrAdd(0x20, 0x0002, 0x51, 1, 0x34, 0x20, 0, 2,
			3, 1, 0x67, 0x88, 2, 1, 0xC00F,
			0xC07F, 0x3838, 0, 4, 0, 0,
			1, 0, 0, 0, 0, 0, 3, 0,
			0, 0x0D, 0x10, 0x0C, 0x0F, 0x0E, 0x0D, 0,
			0, 0, 0, 0, 0, 0, 0, 0xC9,
			sensorName2)

		// sensor 3 - current
		sensorAdd(0x20, 0, 3, 3, 1)
		sensorName3 := []uint8{'M', 'X', 'c', 'u', 'r', 'r', 'e', 'n', 't'}
		mainSdrAdd(0x20, 0x0003, 0x51, 1, 0x34, 0x20, 0, 3,
			3, 1, 0x67, 0x88, 3, 1, 0xC00F,
			0xC07F, 0x3838, 0, 5, 0, 0,
			1, 0, 0, 0, 0, 0, 3, 0,
			0, 3, 6, 5, 7, 6, 5, 0,
			0, 0, 0, 0, 0, 0, 0, 0xC9,
			sensorName3)

		// sensor 4 - fan
		sensorAdd(0x20, 0, 4, 4, 1)
		sensorName4 := []uint8{'F', 'X', 'f', 'a', 'n', 'r', 'e', 'a', 'd'}
		mainSdrAdd(0x20, 0x0004, 0x51, 1, 0x34, 0x20, 0, 4,
			3, 1, 0x67, 0x88, 4, 1, 0xC00F,
			0xC07F, 0x3838, 4, 0x12, 0x0A, 0,
			1, 0, 0, 0, 0, 0, 3, 0,
			0, 0x28, 0x50, 0x32, 0x46, 0x3C, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0xC9,
			sensorName4)
	} else {
		// Do a dynamic discovery of sensors based on sysclass
		// filesystem nodes.
		// Current env chips:
		//ltc4215 (hot-swap controller) ??
		//ucd9090 (voltage/fan/temp monitor)
		//lm75 (temp monitor)
	}

}

func sensorAdd(bmc uint8, lun uint8, num uint8, stype uint8, code uint8) {
	sensor := new(sensorT)
	sensor.mc = bmc
	sensor.lun = lun
	sensor.num = num
	sensor.sensorType = stype
	sensor.eventReadingCode = code
	mc.sensors[lun][num] = sensor

	sensor.enabled = true
	sensor.eventStatus = 0
	sensor.eventsEnabled = true
	sensor.scanningEnabled = true
}

func mainSdrAdd(bmc uint8, recordId uint16, sdrVers uint8,
	recordType uint8, recordLength uint8, sensorOwnerId uint8,
	sensorOwnerLun uint8, sensorNum uint8, entityId uint8,
	entityInstance uint8, sensorInit uint8, sensorCaps uint8,
	sensorType uint8, eventCode uint8, assLthrMask uint16,
	deassHthrMask uint16, drStRdMask uint16, sensorUnits1 uint8,
	sensorUnits2 uint8, sensorUnits3 uint8, linear uint8,
	m uint8, mTol uint8, b uint8, bAcc uint8, accDir uint8,
	rexpBexp uint8, anlogFlags uint8, nomReading uint8,
	normMax uint8, normMin uint8, sensorMax uint8, sensorMin uint8,
	upperNcThr uint8, upperCrThr uint8, upperNcrThr uint8,
	lowerNrThr uint8, lowerCrThr uint8, lowerNcrThr uint8,
	pgThrHyst uint8, ngThrHyst uint8, res1 uint8, res2 uint8,
	oem uint8, idStrLghtCode uint8, idStr []uint8) {

	// Range check the list
	if mc.mainSdrs.nextFreeEntryId == 0xffff {
		fmt.Println("main_sdrs are full!")
		return
	}

	// Obtain and initialize new sdr entry
	newSdr := new(sdrT)
	mc.mainSdrs.nextFreeEntryId++
	newSdr.recordId = mc.mainSdrs.nextFreeEntryId
	newSdr.length = 48 + uint8(idStrLghtCode&0x1F)

	// Serialize SDR record into newSdr's data
	binary.LittleEndian.PutUint16(newSdr.data[0:2], newSdr.recordId)
	newSdr.data[2] = sdrVers
	newSdr.data[3] = recordType
	newSdr.data[4] = recordLength
	newSdr.data[5] = sensorOwnerId
	newSdr.data[6] = sensorOwnerLun
	newSdr.data[7] = sensorNum
	newSdr.data[8] = entityId
	newSdr.data[9] = entityInstance
	newSdr.data[10] = sensorInit
	newSdr.data[11] = sensorCaps
	newSdr.data[12] = sensorType
	newSdr.data[13] = eventCode
	binary.LittleEndian.PutUint16(newSdr.data[14:16], assLthrMask)
	binary.LittleEndian.PutUint16(newSdr.data[16:18], deassHthrMask)
	binary.LittleEndian.PutUint16(newSdr.data[18:20], drStRdMask)
	newSdr.data[20] = sensorUnits1
	newSdr.data[21] = sensorUnits2
	newSdr.data[22] = sensorUnits3
	newSdr.data[23] = linear
	newSdr.data[24] = m
	newSdr.data[25] = mTol
	newSdr.data[26] = b
	newSdr.data[27] = bAcc
	newSdr.data[28] = accDir
	newSdr.data[29] = rexpBexp
	newSdr.data[30] = anlogFlags
	newSdr.data[31] = nomReading
	newSdr.data[32] = normMax
	newSdr.data[33] = normMin
	newSdr.data[34] = sensorMax
	newSdr.data[35] = sensorMin
	newSdr.data[36] = upperNcThr
	newSdr.data[37] = upperCrThr
	newSdr.data[38] = upperNcrThr
	newSdr.data[39] = lowerNrThr
	newSdr.data[40] = lowerCrThr
	newSdr.data[41] = lowerNcrThr
	newSdr.data[42] = pgThrHyst
	newSdr.data[43] = ngThrHyst
	newSdr.data[44] = res1
	newSdr.data[45] = res2
	newSdr.data[46] = oem
	newSdr.data[47] = idStrLghtCode
	idStrLength := idStrLghtCode & 0x1F
	copy(newSdr.data[48:], idStr[0:idStrLength])

	// Add new entry into main_sdr at the tail
	if mc.mainSdrs.sdrs == nil {
		mc.mainSdrs.sdrs = newSdr
	} else {
		mc.mainSdrs.tailSdr.next = newSdr
	}
	mc.mainSdrs.tailSdr = newSdr
	mc.mainSdrs.sdrCount++

	// Update time fields

	// Add support for persistence database
}
