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
- Given the static nature of a white-box networking switch,
  static initialization of several components will be done.
  This is a similar approach to ipmi_sim. These initialized components
  include:
	- username/password and other user parameters
	- sensors and their sdrs (since the switch hw config will be fixed)
  In ipmi_sim, these parameters are specified in lan.conf and .emu files
- Whenever possible use bmc-originated events to a remote controller
  to avoid a polling regimen. This will aid in keeping remote controller
  and network load to a minimum in the context of large data-centers with 
  many switches. This is the so-called push scheme. Of course, with the 
  support of IPMI protocol, remote controllers can pull switch data at any time.

Todo (in priority order):
- Sensor polling support from target sysclass fs
- Logging support (SEL) - can we replace with remote syslog ?
- LAN alerts via PET ? snmpd ?
- Other functions required for white box switch eg cold-reset,
  warm-reset, manufacturing-test
- Internal syslog support (package log/syslog integration)
- Add timestamp support for sdrs
- Persistence support? (at the least some historical record of readings)
  (this is not part of IPMI so this could be displayed via http)
- Straight authentication functionality? IPMI also offers MD2 and MD5
  but both of these are not considered secure. It is debatable whether
  straight password offers any better security for IPMI sessions. It
  is probably best to ensure security by physical access means i.e
  make sure ipmi lan segments are not exposed outside of trusted networks.
- Daemonize ipmigod with double-fork (should be in goes) [done]
- Simulation vs real-target flags [done]
- ASF ping support [done]
- Implement other functions ? (chassis set etc)