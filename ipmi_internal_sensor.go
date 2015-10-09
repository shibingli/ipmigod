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

type sensor_t struct {
	num              uint8
	lun              uint8
	scanning_enabled bool
	events_enabled   bool
	enabled          bool
	mc               uint8

	sensor_type        uint8
	event_reading_code uint8

	value uint8

	hysteresis_support  uint8
	positive_hysteresis uint8
	negative_hysteresis uint8

	threshold_support   uint8
	threshold_supported uint16
	thresholds          [6]uint8

	event_support uint8
	// 0 for assertion, 1 for deassertion.
	event_supported [2]uint16
	event_enabled   [2]uint16

	// Current bit values
	event_status uint16
}

func set_event_receiver(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func get_event_receiver(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func platform_event(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func get_pef_capabilities(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func arm_pef_postpone_timer(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func set_pef_config_parms(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func get_pef_config_parms(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func set_last_processed_event_id(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func get_last_processed_event_id(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func alert_immediate(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func pet_acknowledge(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func get_device_sdr_info(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func get_device_sdr(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func reserve_device_sdr_repository(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func get_sensor_reading_factors(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func set_sensor_hysteresis(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func get_sensor_hysteresis(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func set_sensor_threshold(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func get_sensor_threshold(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func set_sensor_event_enable(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func get_sensor_event_enable(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func rearm_sensor_events(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func get_sensor_event_status(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func get_sensor_reading(msg *msg_t) {

	var data [5]uint8

	sens_num := msg.data[msg.data_start]
	if mc.sensors[msg.rmcp.message.rs_lun][sens_num] == nil {
		msg.return_err(nil, 0x81)
		return
	}

	sensor := mc.sensors[msg.rmcp.message.rs_lun][sens_num]

	data[0] = 0
	if simulate {
		switch sens_num {
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
	var sens_en uint8
	if sensor.events_enabled {
		sens_en = 1
	} else {
		sens_en = 0
	}
	var scan_en uint8
	if sensor.scanning_enabled && sensor.enabled {
		scan_en = 1
	} else {
		scan_en = 0
	}
	data[2] = (sens_en << 7) | (scan_en << 6)
	binary.LittleEndian.PutUint16(data[3:5], sensor.event_status)

	msg.return_rsp_data(nil, data[0:5], 5)
}

func set_sensor_type(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}

func get_sensor_type(msg *msg_t) {
	fmt.Println("sensor_event_netfn not supported",
		msg.rmcp.message.cmd)
}
