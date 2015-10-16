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

	var data [5]uint8

	sensNum := msg.data[msg.dataStart]
	if mc.sensors[msg.rmcp.message.rsLun][sensNum] == nil {
		msg.returnErr(nil, 0x81)
		return
	}

	sensor := mc.sensors[msg.rmcp.message.rsLun][sensNum]

	data[0] = 0
	if simulate {
		switch sensNum {
		case 1:
			data[1] = uint8(rand.Int()&0xf) + 20
		case 2:
			data[1] = uint8(rand.Int() & 0xf)
		case 3:
			data[1] = uint8(rand.Int() & 0x7)
		case 4:
			data[1] = uint8(rand.Int() & 0x3f)
		}
	} else {
		data[1] = sensor.value
	}
	var sensEn uint8
	if sensor.eventsEnabled {
		sensEn = 1
	} else {
		sensEn = 0
	}
	var scanEn uint8
	if sensor.scanningEnabled && sensor.enabled {
		scanEn = 1
	} else {
		scanEn = 0
	}
	data[2] = (sensEn << 7) | (scanEn << 6)
	binary.LittleEndian.PutUint16(data[3:5], sensor.eventStatus)

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
