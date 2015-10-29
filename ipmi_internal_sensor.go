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

type sensorT struct {
	num             uint8
	lun             uint8
	scanningEnabled bool
	eventsEnabled   bool
	enabled         bool
	mc              uint8

	sensorType       uint8
	eventReadingCode uint8

	value uint8

	hysteresisSupport  uint8
	positiveHysteresis uint8
	negativeHysteresis uint8

	thresholdSupport   uint8
	thresholdSupported uint16
	thresholds         [6]uint8

	eventSupport uint8
	// 0 for assertion, 1 for deassertion.
	eventSupported [2]uint16
	eventEnabled   [2]uint16

	// Current bit values
	eventStatus uint16
}

func setEventReceiver(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func getEventReceiver(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func platformEvent(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func getPefCapabilities(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func armPefPostponeTimer(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func setPefConfigParms(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func getPefConfigParms(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func setLastProcessedEventId(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func getLastProcessedEventId(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func alertImmediate(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func petAcknowledge(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func getDeviceSdrInfo(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func getDeviceSdr(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func reserveDeviceSdrRepository(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func getSensorReadingFactors(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func setSensorHysteresis(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func getSensorHysteresis(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func setSensorThreshold(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func getSensorThreshold(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func setSensorEventEnable(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func getSensorEventEnable(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func rearmSensorEvents(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func getSensorEventStatus(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func getSensorReading(msg *msgT) {

	var (
		entry *sdrT
		data  [5]uint8
	)

	sensNum := msg.data[msg.dataStart]
	entry = mc.mainSdrs.sdrs
	for entry != nil {
		if entry.lun == msg.rmcp.message.rsLun &&
			entry.sensNum == sensNum {
			break
		}
		entry = entry.next
	}
	if entry == nil {
		fmt.Printf("getSensorReading: Can't find sensor %d:%d",
			msg.rmcp.message.rsLun, sensNum)
		msg.returnErr(nil, 0x81)
		return
	}

	data[0] = 0
	data[1] = entry.value
	var sensEn uint8
	if entry.eventsEnabled {
		sensEn = 1
	} else {
		sensEn = 0
	}
	var scanEn uint8
	if entry.scanningEnabled && entry.enabled {
		scanEn = 1
	} else {
		scanEn = 0
	}
	data[2] = (sensEn << 7) | (scanEn << 6)
	binary.LittleEndian.PutUint16(data[3:5], entry.eventStatus)

	msg.returnRspData(nil, data[0:5], 5)
}

func setSensorType(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func getSensorType(msg *msgT) {
	fmt.Println("sensorEventNetfn not supported",
		msg.rmcp.message.cmd)
}

func pollSensors() {

	var (
		value   uint8
		lun     uint8
		sensNum uint8
		entry   *sdrT
		msgData []uint8
		try     int
	)
	for lun = 0; lun < 4; lun++ {
		for sensNum = 1; sensNum < 255; sensNum++ {
			if mc.sensors[lun][sensNum] == nil {
				continue
			}
			sensor := mc.sensors[lun][sensNum]

			if simulate {
				// update sensors locally only
				switch sensNum {
				case 1, 17, 33:
					value = uint8(rand.Int()&0xf) + 20
				case 2, 18, 34:
					value = uint8(rand.Int() & 0xf)
				case 3, 19, 35:
					value = uint8(rand.Int() & 0x7)
				case 4, 20, 36:
					value = uint8(rand.Int() & 0x3f)
				}
			} else {
				// FIXME - need to go out to sysfs for value
				// for local sensors. If sensNum >=16 then
				// we are on LC and need to initiate
				// partialAddSdr to MM
			}
			sensor.value = value

			// Find and update SDR for this sensor
			entry = mc.mainSdrs.sdrs
			for entry != nil {
				if entry.lun == lun &&
					entry.sensNum == sensNum {
					break
				}
				entry = entry.next
			}
			entry.value = value

			// If we are on LC and need to initiate
			// special addSdr to MM. This special addSdr will use
			// oem field of SDR record to sneak out the
			// sensor reading for this sensor.
			if chassisCardNum > 0 {
				msgData = addSdrBuildMsg(entry)
				msgData[66] = value
				for try = 0; try < MAX_RETRIES; try++ {
					if ipmiReqRsp(mc.mmConn, msgData,
						addSdrParseRsp) {
						break
					}
				}
				if try >= MAX_RETRIES {
					panic("ipmiClient spec add-sdr to MM")
				}
			}
		}
	}
}
