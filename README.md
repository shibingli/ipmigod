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
  support of IPMI protocol, remote controllers can pull switch data at any 
  time.
- Distributed implememntation for IPMI
  
                           +--------+
                           | MM-BMC |
                           |        |
                           +--------+
                             |    |
                             /    \
                            /      \
                      +--------+  +--------+
                      | LC1-BMC|  | LC2-BMC|
                      |        |  |        |
                      +--------+  +--------+

      MM-BMC is the central aggregator of IPMI state for linecard BMCs
      in a chassis. When SDRs or SELs are created on a linecard, add
      transaction IPMI messages are sent to the MM so it can reflect
      the new state. Periodic poll routines at the LCs will push new
      values to MM.
      In this way, external remote agents can deal with just the MM IPMI
      entity to control/interrogate state for the whole chassis.
      For convenience, it is also possible for the remote agent to deal
      directly with each LC's IPMI implementation to query that LC's
      sensors.

      Taking SDRs as an example - when LC1 creates an SDR for a sensor,
      LC1 will send an add-sdr message to MM which will create an SDR that
      proxies for LC1's sensor. When LC1 updates that sensor with a timed
      poll, a message is sent to MM to update the SDR representing that
      sensor.
 
Todo (in priority order):
- Sensor polling support from target sysclass fs
  	 (simulate inline)       [done]
  	 (simulate with files)
	 (on real hw from sysfs)
- LAN alerts via PET ? snmpd ?
- Other functions required for white box switch eg cold-reset,
  warm-reset, manufacturing-test
- Distribution of SELs
- Distribution of SDRs and sensor readings between LC and MM [done]
- Add timestamp support for sdrs [done]
- Logging support (SEL) [done]
- Persistence support? (at the least some historical record of readings)
  (this is not part of IPMI so this could be displayed via http) [defer]
- Straight authentication functionality? IPMI also offers MD2 and MD5
  but both of these are not considered secure. It is debatable whether
  straight password offers any better security for IPMI sessions. It
  is probably best to ensure security by physical access means i.e
  make sure ipmi lan segments are not exposed outside of 
  trusted networks. [defer]
- Daemonize ipmigod with double-fork (should be in goes) [done]
- Simulation vs real-target flags [done]
- ASF ping support [done]
- Implement other functions ? (chassis set etc)
