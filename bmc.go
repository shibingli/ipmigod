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

type sdr_t struct {
	record_id uint16
	length    uint8
	data      [76]uint8
	next      *sdr_t
}

type sdrs_t struct {
	reservation        uint16
	sdr_count          uint16
	sensor_count       uint16
	last_add_time      uint32
	last_erase_time    uint32
	time_offset        uint64
	flags              uint8
	next_free_entry_id uint16
	sdrs_length        uint32
	tail_sdr           *sdr_t
	sdrs               *sdr_t // A linked list of SDR entries.

}

type mc_t struct {
	bmc_ipmb        uint8 // address of bmc
	device_id       uint8
	has_device_sdrs bool
	device_revision uint8
	major_fw_rev    uint8
	minor_fw_rev    uint8
	device_support  uint8
	mfg_id          [3]uint8
	product_id      [2]uint8
	main_sdrs       sdrs_t
	sensors         [4][255]*sensor_t
}

var mc mc_t

func bmc_init() {
	// Initialize the bmc
	mc.bmc_ipmb = 0x20
	mc.device_id = 0
	mc.has_device_sdrs = false
	mc.device_revision = 1
	mc.major_fw_rev = 1
	mc.minor_fw_rev = 1
	mc.device_support = IPMI_DEVID_SDR_REPOSITORY_DEV |
		IPMI_DEVID_SENSOR_DEV
	mc.mfg_id[0] = 0
	mc.mfg_id[1] = 0
	mc.mfg_id[2] = 1
	mc.product_id[0] = 0
	mc.product_id[1] = 0

	mc.main_sdrs.flags = IPMI_SDR_RESERVE_SDR_SUPPORTED

	// Initially this is a simulated set of sensors.
	// In production, a similar scheme could be used or
	// perhaps a more dynamic scheme where the sysclass fs is
	// scanned to gather these params.
	if simulate {
		// sensor 1 - temp
		sensor_add(0x20, 0, 1, 1, 1)
		sensor_name1 := []uint8{'D', 'J', 't', 'e', 'm', 'p'}
		main_sdr_add(0x20, 0x0001, 0x51, 1, 0x31, 0x20, 0, 1,
			3, 1, 0x67, 0x88, 1, 1, 0xC00F,
			0xC07F, 0x3838, 0, 1, 0, 0,
			1, 0, 0, 0, 0, 0, 3, 0x60,
			0xB0, 0, 0xB0, 0, 0xA0, 0x90, 0x66, 0,
			0, 0, 0, 0, 0, 0, 0, 0xC6,
			sensor_name1)

		// sensor 2 - voltage
		sensor_add(0x20, 0, 2, 2, 1)
		sensor_name2 := []uint8{'M', 'X', 'v', 'o', 'l', 't', 'a', 'g', 'e'}
		main_sdr_add(0x20, 0x0002, 0x51, 1, 0x34, 0x20, 0, 2,
			3, 1, 0x67, 0x88, 2, 1, 0xC00F,
			0xC07F, 0x3838, 0, 4, 0, 0,
			1, 0, 0, 0, 0, 0, 3, 0,
			0, 0x0D, 0x10, 0x0C, 0x0F, 0x0E, 0x0D, 0,
			0, 0, 0, 0, 0, 0, 0, 0xC9,
			sensor_name2)

		// sensor 3 - current
		sensor_add(0x20, 0, 3, 3, 1)
		sensor_name3 := []uint8{'M', 'X', 'c', 'u', 'r', 'r', 'e', 'n', 't'}
		main_sdr_add(0x20, 0x0003, 0x51, 1, 0x34, 0x20, 0, 3,
			3, 1, 0x67, 0x88, 3, 1, 0xC00F,
			0xC07F, 0x3838, 0, 5, 0, 0,
			1, 0, 0, 0, 0, 0, 3, 0,
			0, 3, 6, 5, 7, 6, 5, 0,
			0, 0, 0, 0, 0, 0, 0, 0xC9,
			sensor_name3)

		// sensor 4 - fan
		sensor_add(0x20, 0, 4, 4, 1)
		sensor_name4 := []uint8{'F', 'X', 'f', 'a', 'n', 'r', 'e', 'a', 'd'}
		main_sdr_add(0x20, 0x0004, 0x51, 1, 0x34, 0x20, 0, 4,
			3, 1, 0x67, 0x88, 4, 1, 0xC00F,
			0xC07F, 0x3838, 4, 0x12, 0x0A, 0,
			1, 0, 0, 0, 0, 0, 3, 0,
			0, 0x28, 0x50, 0x32, 0x46, 0x3C, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0xC9,
			sensor_name4)
	}

}

func sensor_add(bmc uint8, lun uint8, num uint8, stype uint8, code uint8) {
	sensor := new(sensor_t)
	sensor.mc = bmc
	sensor.lun = lun
	sensor.num = num
	sensor.sensor_type = stype
	sensor.event_reading_code = code
	mc.sensors[lun][num] = sensor

	sensor.enabled = true
	sensor.event_status = 0
	sensor.events_enabled = true
	sensor.scanning_enabled = true
}

func main_sdr_add(bmc uint8, record_id uint16, sdr_vers uint8,
	record_type uint8, record_length uint8, sensor_owner_id uint8,
	sensor_owner_lun uint8, sensor_num uint8, entity_id uint8,
	entity_instance uint8, sensor_init uint8, sensor_caps uint8,
	sensor_type uint8, event_code uint8, ass_lthr_mask uint16,
	deass_hthr_mask uint16, dr_st_rd_mask uint16, sensor_units1 uint8,
	sensor_units2 uint8, sensor_units3 uint8, linear uint8,
	m uint8, m_tol uint8, b uint8, b_acc uint8, acc_dir uint8,
	rexp_bexp uint8, anlog_flags uint8, nom_reading uint8,
	norm_max uint8, norm_min uint8, sensor_max uint8, sensor_min uint8,
	upper_nc_thr uint8, upper_cr_thr uint8, upper_ncr_thr uint8,
	lower_nr_thr uint8, lower_cr_thr uint8, lower_ncr_thr uint8,
	pg_thr_hyst uint8, ng_thr_hyst uint8, res1 uint8, res2 uint8,
	oem uint8, id_str_lght_code uint8, id_str []uint8) {

	// Range check the list
	if mc.main_sdrs.next_free_entry_id == 0xffff {
		fmt.Println("main_sdrs are full!")
		return
	}

	// Obtain and initialize new sdr entry
	new_sdr := new(sdr_t)
	mc.main_sdrs.next_free_entry_id++
	new_sdr.record_id = mc.main_sdrs.next_free_entry_id
	new_sdr.length = 48 + uint8(id_str_lght_code&0x1F) // + 6

	// Serialize SDR record into new_sdr's data
	binary.LittleEndian.PutUint16(new_sdr.data[0:2], new_sdr.record_id)
	new_sdr.data[2] = sdr_vers
	new_sdr.data[3] = record_type
	new_sdr.data[4] = record_length
	new_sdr.data[5] = sensor_owner_id
	new_sdr.data[6] = sensor_owner_lun
	new_sdr.data[7] = sensor_num
	new_sdr.data[8] = entity_id
	new_sdr.data[9] = entity_instance
	new_sdr.data[10] = sensor_init
	new_sdr.data[11] = sensor_caps
	new_sdr.data[12] = sensor_type
	new_sdr.data[13] = event_code
	binary.LittleEndian.PutUint16(new_sdr.data[14:16], ass_lthr_mask)
	binary.LittleEndian.PutUint16(new_sdr.data[16:18], deass_hthr_mask)
	binary.LittleEndian.PutUint16(new_sdr.data[18:20], dr_st_rd_mask)
	new_sdr.data[20] = sensor_units1
	new_sdr.data[21] = sensor_units2
	new_sdr.data[22] = sensor_units3
	new_sdr.data[23] = linear
	new_sdr.data[24] = m
	new_sdr.data[25] = m_tol
	new_sdr.data[26] = b
	new_sdr.data[27] = b_acc
	new_sdr.data[28] = acc_dir
	new_sdr.data[29] = rexp_bexp
	new_sdr.data[30] = anlog_flags
	new_sdr.data[31] = nom_reading
	new_sdr.data[32] = norm_max
	new_sdr.data[33] = norm_min
	new_sdr.data[34] = sensor_max
	new_sdr.data[35] = sensor_min
	new_sdr.data[36] = upper_nc_thr
	new_sdr.data[37] = upper_cr_thr
	new_sdr.data[38] = upper_ncr_thr
	new_sdr.data[39] = lower_nr_thr
	new_sdr.data[40] = lower_cr_thr
	new_sdr.data[41] = lower_ncr_thr
	new_sdr.data[42] = pg_thr_hyst
	new_sdr.data[43] = ng_thr_hyst
	new_sdr.data[44] = res1
	new_sdr.data[45] = res2
	new_sdr.data[46] = oem
	new_sdr.data[47] = id_str_lght_code
	id_str_length := id_str_lght_code & 0x1F
	copy(new_sdr.data[48:], id_str[0:id_str_length])

	// Add new entry into main_sdr at the tail
	if mc.main_sdrs.sdrs == nil {
		mc.main_sdrs.sdrs = new_sdr
	} else {
		mc.main_sdrs.tail_sdr.next = new_sdr
	}
	mc.main_sdrs.tail_sdr = new_sdr
	mc.main_sdrs.sdr_count++

	// Update time fields

	// Add support for persistence database
}
