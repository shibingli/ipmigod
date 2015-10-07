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

func get_fru_inventory_area_info(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func read_fru_data(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func write_fru_data(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func get_sdr_repository_info(msg *msg_t) {
	var data [15]uint8

	data[0] = 0
	data[1] = 0x51
	binary.LittleEndian.PutUint16(data[2:4], mc.main_sdrs.sdr_count)
	space := MAX_SDR_LENGTH * (MAX_NUM_SDRS - mc.main_sdrs.sdr_count)
	if space > 0xfffe {
		space = 0xfffe
	}
	binary.LittleEndian.PutUint16(data[4:6], space)
	binary.LittleEndian.PutUint32(data[6:10],
		mc.main_sdrs.last_add_time)
	binary.LittleEndian.PutUint32(data[10:14],
		mc.main_sdrs.last_erase_time)
	data[14] = mc.main_sdrs.flags

	msg.return_rsp_data(nil, data[0:15], 15)
}

func get_sdr_repository_alloc_info(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func reserve_sdr_repository(msg *msg_t) {
	var data [3]uint8

	mc.main_sdrs.reservation++
	if mc.main_sdrs.reservation == 0 {
		mc.main_sdrs.reservation++
	}

	data[0] = 0
	binary.LittleEndian.PutUint16(data[1:3], mc.main_sdrs.reservation)

	msg.return_rsp_data(nil, data[0:3], 3)
}

func get_sdr(msg *msg_t) {

	var (
		data  [MAX_MSG_RETURN_DATA]uint8
		entry *sdr_t
	)

	data_start := msg.data_start
	reservation :=
		binary.LittleEndian.Uint16(msg.data[data_start : data_start+2])

	if reservation != 0 && reservation != mc.main_sdrs.reservation {
		fmt.Println("get_sdr: reservation mismatch", reservation,
			mc.main_sdrs.reservation)
		msg.return_err(nil, IPMI_INVALID_RESERVATION_CC)
		return
	}

	record_id :=
		binary.LittleEndian.Uint16(msg.data[data_start+2 : data_start+4])
	offset := msg.data[data_start+4]
	count := msg.data[data_start+5]

	if record_id == 0 {
		entry = mc.main_sdrs.sdrs
	} else if record_id == 0xffff {
		entry = mc.main_sdrs.tail_sdr
	} else {
		entry = mc.main_sdrs.sdrs
		for entry != nil {
			if entry.record_id == record_id {
				break
			}
			entry = entry.next
		}
	}

	if entry == nil {
		fmt.Println("get_sdr: Can't find record_id",
			record_id)
		msg.return_err(nil, IPMI_NOT_PRESENT_CC)
		return
	}

	if offset >= entry.length {
		fmt.Println("get_sdr: offset out of range")
		msg.return_err(nil, IPMI_PARAMETER_OUT_OF_RANGE_CC)
		return
	}

	if (offset + count) > entry.length {
		count = entry.length - offset
	}
	if uint(count+3) > MAX_MSG_RETURN_DATA {
		fmt.Println("get_sdr: cannot return required data")
		// Too much data to put into response.
		msg.return_err(nil,
			IPMI_CANNOT_RETURN_REQ_LENGTH_CC)
		return
	}

	data[0] = 0
	if entry.next != nil {
		binary.LittleEndian.PutUint16(data[1:3],
			entry.next.record_id)
	} else {
		data[1] = 0xff
		data[2] = 0xff
	}

	copy(data[3:], entry.data[offset:offset+count])
	msg.return_rsp_data(nil, data[0:], uint(count+3))
}

func add_sdr_cmd(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func partial_add_sdr(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func delete_sdr(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func clear_sdr_repository(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func get_sdr_repository_time(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func set_sdr_repository_time(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func enter_sdr_repository_update(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func exit_sdr_repository_update(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func run_initialization_agent(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func get_sel_info(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func get_sel_allocation_info(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func reserve_sel(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func get_sel_entry(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func add_sel_entry(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func partial_add_sel_entry(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func delete_sel_entry(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func clear_sel(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func get_sel_time(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func set_sel_time(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func get_auxiliary_log_status(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}

func set_auxiliary_log_status(msg *msg_t) {
	fmt.Println("storage_netfn not supported", msg.rmcp.message.cmd)
}
