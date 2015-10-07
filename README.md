# ipmigod
IPMI protocol implementation written in go

Notes:
- Networks communication between freeipmi utils and openipmi
  are in LittleEndian format. To adhere to this implementation,
  this daemon sends data over the network in LittleEndian (against
  all instincts :)

Architecture Notes:

- A limited number of netfunctions/commands are implemented
  with a mind to having the minimal set required to control
  a white-box networking switch over lan
- Given the static nature of a white-box networking switch
  static initialization of several components will be done.
  This is a similar approach to ipmi_sim. These initialized components
  include:
	- username/password and other user parameters
	- sensors and their sdrs (since the switch hw config will be fixed)
  In ipmi_sim, these parameters are specified in lan.conf and .emu files
- Whenever possible use bmc-originated events to a remote controller
  to avoid a polling regimen. This will aid keeping remote controller
  load to a minimum in the context of large data-centers with many switches.
  This is the so-called push scheme. Of course, with the support of IPMI
  protocol, remote controllers can pull switch data at any time.

Todo:
- Simulation vs real-target flags
- Sensor polling support from target sysclass fs
- Straight authentication
- Persistence support