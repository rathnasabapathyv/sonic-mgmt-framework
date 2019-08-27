package translib

import (
	"errors"
	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
	"net"
	"strconv"
	"translib/db"
	"translib/ocbinds"
	"translib/tlerr"
	"unsafe"
)

const (
	PORT         = "PORT"
	INDEX        = "index"
	MTU          = "mtu"
	ADMIN_STATUS = "admin_status"
	SPEED        = "speed"
	DESC         = "description"
	OPER_STATUS  = "oper_status"
)

type Table int

const (
	IF_TABLE_MAP Table = iota
	PORT_STAT_MAP
)

func (app *IntfApp) translateCommonIntfConfig(ifKey *string, intf *ocbinds.OpenconfigInterfaces_Interfaces_Interface, curr *db.Value) {
	if intf.Config == nil {
		return
	}
	if intf.Config.Description != nil {
		log.Info("Description = ", *intf.Config.Description)
		curr.Field["description"] = *intf.Config.Description
	} else if intf.Config.Mtu != nil {
		log.Info("mtu:= ", *intf.Config.Mtu)
		curr.Field["mtu"] = strconv.Itoa(int(*intf.Config.Mtu))
	} else if intf.Config.Enabled != nil {
		log.Info("enabled = ", *intf.Config.Enabled)
		if *intf.Config.Enabled == true {
			curr.Field["admin_status"] = "up"
		} else {
			curr.Field["admin_status"] = "down"
		}
	}
	log.Info("Writing to db for ", *ifKey)
	app.ifTableMap[*ifKey] = dbEntry{op: opUpdate, entry: *curr}
}

func (app *IntfApp) processGetSpecificIntf(dbs [db.MaxDB]*db.DB, targetUriPath *string) (GetResponse, error) {
	var err error
	var payload []byte
	pathInfo := app.path
	intfObj := app.getAppRootObject()

	log.Infof("Received GET for path %s; template: %s vars=%v", pathInfo.Path, pathInfo.Template, pathInfo.Vars)

	if intfObj.Interface != nil && len(intfObj.Interface) > 0 {
		/* Interface name is the key */
		for ifKey, _ := range intfObj.Interface {
			log.Info("Interface Name = ", ifKey)
			ifInfo := intfObj.Interface[ifKey]
			/* Filling Interface Info to internal DS */
			err = app.convertDBIntfInfoToInternal(app.appDB, ifKey, db.Key{Comp: []string{ifKey}})
			if err != nil {
				return GetResponse{Payload: payload, ErrSrc: AppErr}, err
			}

			/*Check if the request is for a specific attribute in Interfaces state container*/
			oc_val := &ocbinds.OpenconfigInterfaces_Interfaces_Interface_State{}
			ok, e := app.getSpecificAttr(*targetUriPath, ifKey, oc_val)
			if ok {
				if e != nil {
					return GetResponse{Payload: payload, ErrSrc: AppErr}, e
				}

				payload, err = dumpIetfJson(oc_val, false)
				if err == nil {
					return GetResponse{Payload: payload}, err
				} else {
					return GetResponse{Payload: payload, ErrSrc: AppErr}, err
				}
			}

			/* Filling the counter Info to internal DS */
			err = app.getPortOidMapForCounters(app.countersDB)
			if err != nil {
				return GetResponse{Payload: payload, ErrSrc: AppErr}, err
			}
			err = app.convertDBIntfCounterInfoToInternal(app.countersDB, ifKey)
			if err != nil {
				return GetResponse{Payload: payload, ErrSrc: AppErr}, err
			}

			/*Check if the request is for a specific attribute in Interfaces state COUNTERS container*/
			counter_val := &ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_Counters{}
			ok, e = app.getSpecificCounterAttr(*targetUriPath, ifKey, counter_val)
			if ok {
				if e != nil {
					return GetResponse{Payload: payload, ErrSrc: AppErr}, e
				}

				payload, err = dumpIetfJson(counter_val, false)
				if err == nil {
					return GetResponse{Payload: payload}, err
				} else {
					return GetResponse{Payload: payload, ErrSrc: AppErr}, err
				}
			}

			/* Filling Interface IP info to internal DS */
			err = app.convertDBIntfIPInfoToInternal(app.appDB, ifKey)
			if err != nil {
				return GetResponse{Payload: payload, ErrSrc: AppErr}, err
			}

			/* Filling the tree with the info we have in Internal DS */
			ygot.BuildEmptyTree(ifInfo)
			if *app.ygotTarget == ifInfo.State {
				ygot.BuildEmptyTree(ifInfo.State)
			}
			app.convertInternalToOCIntfInfo(&ifKey, ifInfo)
			if *app.ygotTarget == ifInfo {
				payload, err = dumpIetfJson(intfObj, false)
			} else {
				dummyifInfo := &ocbinds.OpenconfigInterfaces_Interfaces_Interface{}
				if *app.ygotTarget == ifInfo.Config {
					dummyifInfo.Config = ifInfo.Config
					payload, err = dumpIetfJson(dummyifInfo, false)
				} else if *app.ygotTarget == ifInfo.State {
					dummyifInfo.State = ifInfo.State
					payload, err = dumpIetfJson(dummyifInfo, false)
				} else {
					log.Info("Not supported get type!")
					err = errors.New("Requested get-type not supported!")
				}
			}
		}
	}
	return GetResponse{Payload: payload}, err
}

func (app *IntfApp) processGetAllInterfaces(dbs [db.MaxDB]*db.DB) (GetResponse, error) {
	var err error
	var payload []byte
	intfObj := app.getAppRootObject()

	log.Info("Get all Interfaces request!")

	/* Filling Interface Info to internal DS */
	err = app.convertDBIntfInfoToInternal(app.appDB, "", db.Key{})
	if err != nil {
		return GetResponse{Payload: payload, ErrSrc: AppErr}, err
	}
	/* Filling Interface IP info to internal DS */
	err = app.convertDBIntfIPInfoToInternal(app.appDB, "")
	if err != nil {
		return GetResponse{Payload: payload, ErrSrc: AppErr}, err
	}
	/* Filling the counter Info to internal DS */
	err = app.getPortOidMapForCounters(app.countersDB)
	if err != nil {
		return GetResponse{Payload: payload, ErrSrc: AppErr}, err
	}
	err = app.convertDBIntfCounterInfoToInternal(app.countersDB, "")
	if err != nil {
		return GetResponse{Payload: payload, ErrSrc: AppErr}, err
	}
	ygot.BuildEmptyTree(intfObj)
	for ifKey, _ := range app.ifTableMap {
		log.Info("If Key = ", ifKey)
		ifInfo, err := intfObj.NewInterface(ifKey)
		if err != nil {
			log.Errorf("Creation of interface subtree for %s failed!", ifKey)
			return GetResponse{Payload: payload, ErrSrc: AppErr}, err
		}
		ygot.BuildEmptyTree(ifInfo)
		app.convertInternalToOCIntfInfo(&ifKey, ifInfo)
	}
	if *app.ygotTarget == intfObj {
		payload, err = dumpIetfJson((*app.ygotRoot).(*ocbinds.Device), true)
	} else {
		log.Error("Wrong request!")
	}
	return GetResponse{Payload: payload}, err
}

func (app *IntfApp) getSpecificAttr(targetUriPath string, ifKey string, oc_val *ocbinds.OpenconfigInterfaces_Interfaces_Interface_State) (bool, error) {
	switch targetUriPath {
	case "/openconfig-interfaces:interfaces/interface/state/oper-status":
		val, e := app.getIntfAttr(ifKey, OPER_STATUS, IF_TABLE_MAP)
		if len(val) > 0 {
			switch val {
			case "up":
				oc_val.OperStatus = ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_OperStatus_UP
			case "down":
				oc_val.OperStatus = ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_OperStatus_DOWN
			default:
				oc_val.OperStatus = ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_OperStatus_UNSET
			}
			return true, nil
		} else {
			return true, e
		}
	case "/openconfig-interfaces:interfaces/interface/state/admin-status":
		val, e := app.getIntfAttr(ifKey, ADMIN_STATUS, IF_TABLE_MAP)
		if len(val) > 0 {
			switch val {
			case "up":
				oc_val.AdminStatus = ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_AdminStatus_UP
			case "down":
				oc_val.AdminStatus = ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_AdminStatus_DOWN
			default:
				oc_val.AdminStatus = ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_AdminStatus_UNSET
			}
			return true, nil
		} else {
			return true, e
		}
	case "/openconfig-interfaces:interfaces/interface/state/mtu":
		val, e := app.getIntfAttr(ifKey, MTU, IF_TABLE_MAP)
		if len(val) > 0 {
			v, e := strconv.ParseUint(val, 10, 16)
			if e == nil {
				oc_val.Mtu = (*uint16)(unsafe.Pointer(&v))
				return true, nil
			}
		}
		return true, e
	case "/openconfig-interfaces:interfaces/interface/state/ifindex":
		val, e := app.getIntfAttr(ifKey, INDEX, IF_TABLE_MAP)
		if len(val) > 0 {
			v, e := strconv.ParseUint(val, 10, 32)
			if e == nil {
				oc_val.Ifindex = (*uint32)(unsafe.Pointer(&v))
				return true, nil
			}
		}
		return true, e
	case "/openconfig-interfaces:interfaces/interface/state/description":
		val, e := app.getIntfAttr(ifKey, DESC, IF_TABLE_MAP)
		if e == nil {
			oc_val.Description = &val
			return true, nil
		}
		return true, e

	default:
		log.Infof(targetUriPath + " - Not an interface state attribute")
	}
	return false, nil
}

func (app *IntfApp) getSpecificCounterAttr(targetUriPath string, ifKey string, counter_val *ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_Counters) (bool, error) {

	var e error

	switch targetUriPath {
	case "/openconfig-interfaces:interfaces/interface/state/counters/in-octets":
		e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_IN_OCTETS", &counter_val.InOctets)
		return true, e

	case "/openconfig-interfaces:interfaces/interface/state/counters/in-unicast-pkts":
		e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_IN_UCAST_PKTS", &counter_val.InUnicastPkts)
		return true, e

	case "/openconfig-interfaces:interfaces/interface/state/counters/in-broadcast-pkts":
		e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_IN_BROADCAST_PKTS", &counter_val.InBroadcastPkts)
		return true, e

	case "/openconfig-interfaces:interfaces/interface/state/counters/in-multicast-pkts":
		e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_IN_MULTICAST_PKTS", &counter_val.InMulticastPkts)
		return true, e

	case "/openconfig-interfaces:interfaces/interface/state/counters/in-errors":
		e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_IN_ERRORS", &counter_val.InErrors)
		return true, e

	case "/openconfig-interfaces:interfaces/interface/state/counters/in-discards":
		e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_IN_DISCARDS", &counter_val.InDiscards)
		return true, e

	case "/openconfig-interfaces:interfaces/interface/state/counters/in-pkts":
		var inNonUCastPkt, inUCastPkt *uint64
		var in_pkts uint64

		e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_IN_NON_UCAST_PKTS", &inNonUCastPkt)
		if e == nil {
			e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_IN_UCAST_PKTS", &inUCastPkt)
			if e != nil {
				return true, e
			}
			in_pkts = *inUCastPkt + *inNonUCastPkt
			counter_val.InPkts = &in_pkts
			return true, e
		} else {
			return true, e
		}

	case "/openconfig-interfaces:interfaces/interface/state/counters/out-octets":
		e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_OUT_OCTETS", &counter_val.OutOctets)
		return true, e

	case "/openconfig-interfaces:interfaces/interface/state/counters/out-unicast-pkts":
		e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_OUT_UCAST_PKTS", &counter_val.OutUnicastPkts)
		return true, e

	case "/openconfig-interfaces:interfaces/interface/state/counters/out-broadcast-pkts":
		e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_OUT_BROADCAST_PKTS", &counter_val.OutBroadcastPkts)
		return true, e

	case "/openconfig-interfaces:interfaces/interface/state/counters/out-multicast-pkts":
		e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_OUT_MULTICAST_PKTS", &counter_val.OutMulticastPkts)
		return true, e

	case "/openconfig-interfaces:interfaces/interface/state/counters/out-errors":
		e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_OUT_ERRORS", &counter_val.OutErrors)
		return true, e

	case "/openconfig-interfaces:interfaces/interface/state/counters/out-discards":
		e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_OUT_DISCARDS", &counter_val.OutDiscards)
		return true, e

	case "/openconfig-interfaces:interfaces/interface/state/counters/out-pkts":
		var outNonUCastPkt, outUCastPkt *uint64
		var out_pkts uint64

		e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_OUT_NON_UCAST_PKTS", &outNonUCastPkt)
		if e == nil {
			e = app.getCounters(ifKey, "SAI_PORT_STAT_IF_OUT_UCAST_PKTS", &outUCastPkt)
			if e != nil {
				return true, e
			}
			out_pkts = *outUCastPkt + *outNonUCastPkt
			counter_val.OutPkts = &out_pkts
			return true, e
		} else {
			return true, e
		}

	default:
		log.Infof(targetUriPath + " - Not an interface state counter attribute")
	}
	return false, nil
}

func (app *IntfApp) getCounters(ifKey string, attr string, counter_val **uint64) error {
	val, e := app.getIntfAttr(ifKey, attr, PORT_STAT_MAP)
	if len(val) > 0 {
		v, e := strconv.ParseUint(val, 10, 64)
		if e == nil {
			*counter_val = &v
			return nil
		}
	}
	return e
}

func (app *IntfApp) getIntfAttr(ifName string, attr string, table Table) (string, error) {

	var ok bool = false
	var entry dbEntry

	if table == IF_TABLE_MAP {
		entry, ok = app.ifTableMap[ifName]
	} else if table == PORT_STAT_MAP {
		entry, ok = app.intfD.portStatMap[ifName]
	} else {
		return "", errors.New("Unsupported table")
	}

	if ok {
		ifData := entry.entry

		if val, ok := ifData.Field[attr]; ok {
			return val, nil
		}
	}
	return "", errors.New("Attr " + attr + "doesn't exist in IF table Map!")
}

/***********  Translation Helper fn to convert DB Interface info to Internal DS   ***********/
func (app *IntfApp) getPortOidMapForCounters(dbCl *db.DB) error {
	var err error
	ifCountInfo, err := dbCl.GetMapAll(app.intfD.portOidCountrTblTs)
	if err != nil {
		log.Error("Port-OID (Counters) get for all the interfaces failed!")
		return err
	}
	if ifCountInfo.IsPopulated() {
		app.intfD.portOidMap.entry = ifCountInfo
	} else {
		return errors.New("Get for OID info from all the interfaces from Counters DB failed!")
	}
	return err
}

func (app *IntfApp) convertDBIntfCounterInfoToInternal(dbCl *db.DB, ifKey string) error {
	var err error

	if len(ifKey) > 0 {
		oid := app.intfD.portOidMap.entry.Field[ifKey]
		log.Infof("OID : %s received for Interface : %s", oid, ifKey)

		/* Get the statistics for the port */
		var ifStatKey db.Key
		ifStatKey.Comp = []string{oid}

		ifStatInfo, err := dbCl.GetEntry(app.intfD.intfCountrTblTs, ifStatKey)
		if err != nil {
			log.Infof("Fetching port-stat for port : %s failed!", ifKey)
			return err
		}
		app.intfD.portStatMap[ifKey] = dbEntry{entry: ifStatInfo}
	} else {
		for ifKey, _ := range app.ifTableMap {
			app.convertDBIntfCounterInfoToInternal(dbCl, ifKey)
		}
	}
	return err
}

func (app *IntfApp) convertDBIntfInfoToInternal(dbCl *db.DB, ifName string, ifKey db.Key) error {

	var err error
	/* Fetching DB data for a specific Interface */
	if len(ifName) > 0 {

		var ts *db.TableSpec
		switch app.intfType {
		case ETHERNET:
			ts = app.intfD.portTblTs
		case VLAN:
			ts = app.vlanD.vlanTblTs
		}

		log.Info("Updating Interface info from APP-DB to Internal DS for Interface name : ", ifName)
		ifInfo, err := dbCl.GetEntry(ts, ifKey)
		if err != nil {
			log.Errorf("Error found on fetching Interface info from App DB for If Name : %s", ifName)
			errStr := "Invalid Interface:" + ifName
			err = tlerr.InvalidArgsError{Format: errStr}
			return err
		}
		if ifInfo.IsPopulated() {
			log.Info("Interface Info populated for ifName : ", ifName)
			app.ifTableMap[ifName] = dbEntry{entry: ifInfo}
		} else {
			return errors.New("Populating Interface info for " + ifName + "failed")
		}
	} else {
		log.Info("App-DB get for all the interfaces")
		tbl, err := dbCl.GetTable(app.intfD.portTblTs)
		if err != nil {
			log.Error("App-DB get for list of interfaces failed!")
			return err
		}
		keys, _ := tbl.GetKeys()
		for _, key := range keys {
			app.convertDBIntfInfoToInternal(dbCl, key.Get(0), db.Key{Comp: []string{key.Get(0)}})
		}
	}
	return err
}

/***********  Translation Helper fn to convert DB Interface IP info to Internal DS   ***********/
func (app *IntfApp) convertDBIntfIPInfoToInternal(dbCl *db.DB, ifName string) error {

	var err error
	log.Info("Updating Interface IP Info from APP-DB to Internal DS for Interface Name : ", ifName)
	app.allIpKeys, _ = app.doGetAllIpKeys(dbCl, app.intfD.intfIPTblTs)

	for _, key := range app.allIpKeys {
		if len(key.Comp) <= 1 {
			continue
		}
		ipInfo, err := dbCl.GetEntry(app.intfD.intfIPTblTs, key)
		if err != nil {
			log.Errorf("Error found on fetching Interface IP info from App DB for Interface Name : %s", ifName)
			return err
		}
		if len(app.intfD.ifIPTableMap[key.Get(0)]) == 0 {
			app.intfD.ifIPTableMap[key.Get(0)] = make(map[string]dbEntry)
			app.intfD.ifIPTableMap[key.Get(0)][key.Get(1)] = dbEntry{entry: ipInfo}
		} else {
			app.intfD.ifIPTableMap[key.Get(0)][key.Get(1)] = dbEntry{entry: ipInfo}
		}
	}
	return err
}

func (app *IntfApp) convertInternalToOCIntfInfo(ifName *string, ifInfo *ocbinds.OpenconfigInterfaces_Interfaces_Interface) {
	app.convertInternalToOCIntfAttrInfo(ifName, ifInfo)
	app.convertInternalToOCIntfIPAttrInfo(ifName, ifInfo)
	app.convertInternalToOCPortStatInfo(ifName, ifInfo)
}

func (app *IntfApp) convertInternalToOCIntfAttrInfo(ifName *string, ifInfo *ocbinds.OpenconfigInterfaces_Interfaces_Interface) {

	/* Handling the Interface attributes */
	if entry, ok := app.ifTableMap[*ifName]; ok {
		ifData := entry.entry

		name := *ifName
		ifInfo.Config.Name = &name
		ifInfo.State.Name = &name

		for ifAttr := range ifData.Field {
			switch ifAttr {
			case ADMIN_STATUS:
				adminStatus := ifData.Get(ifAttr)
				ifInfo.State.AdminStatus = ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_AdminStatus_DOWN
				if adminStatus == "up" {
					ifInfo.State.AdminStatus = ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_AdminStatus_UP
				}
			case OPER_STATUS:
				operStatus := ifData.Get(ifAttr)
				ifInfo.State.OperStatus = ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_OperStatus_DOWN
				if operStatus == "up" {
					ifInfo.State.OperStatus = ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_OperStatus_UP
				}
			case DESC:
				descVal := ifData.Get(ifAttr)
				descr := new(string)
				*descr = descVal
				ifInfo.Config.Description = descr
				ifInfo.State.Description = descr
			case MTU:
				mtuStr := ifData.Get(ifAttr)
				mtuVal, err := strconv.Atoi(mtuStr)
				mtu := new(uint16)
				*mtu = uint16(mtuVal)
				if err == nil {
					ifInfo.Config.Mtu = mtu
					ifInfo.State.Mtu = mtu
				}
			case SPEED:
				speed := ifData.Get(ifAttr)
				var speedEnum ocbinds.E_OpenconfigIfEthernet_ETHERNET_SPEED

				switch speed {
				case "40000":
					speedEnum = ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_40GB
					speedEnum = ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_40GB
				case "25000":
					speedEnum = ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_25GB
					speedEnum = ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_25GB
				case "10000":
					speedEnum = ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_10GB
					speedEnum = ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_10GB
				case "5000":
					speedEnum = ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_5GB
					speedEnum = ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_5GB
				case "1000":
					speedEnum = ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_1GB
					speedEnum = ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_1GB
				default:
					log.Infof("Not supported speed: %s!", speed)
				}
				ifInfo.Ethernet.Config.PortSpeed = speedEnum
				ifInfo.Ethernet.State.PortSpeed = speedEnum
			case INDEX:
				ifIdxStr := ifData.Get(ifAttr)
				ifIdxNum, err := strconv.Atoi(ifIdxStr)
				if err == nil {
					ifIdx := new(uint32)
					*ifIdx = uint32(ifIdxNum)
					ifInfo.State.Ifindex = ifIdx
				}
			default:
				log.Info("Not a valid attribute!")
			}
		}
	}

}

func (app *IntfApp) convertInternalToOCIntfIPAttrInfo(ifName *string, ifInfo *ocbinds.OpenconfigInterfaces_Interfaces_Interface) {

	/* Handling the Interface IP attributes */
	subIntf, err := ifInfo.Subinterfaces.NewSubinterface(0)
	if err != nil {
		log.Error("Creation of subinterface subtree failed!")
		return
	}
	ygot.BuildEmptyTree(subIntf)
	if ipMap, ok := app.intfD.ifIPTableMap[*ifName]; ok {
		for ipKey, _ := range ipMap {
			log.Info("IP address = ", ipKey)
			ipB, ipNetB, _ := net.ParseCIDR(ipKey)

			v4Flag := false
			v6Flag := false

			var v4Address *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address
			var v6Address *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address

			if validIPv4(ipB.String()) {
				v4Address, err = subIntf.Ipv4.Addresses.NewAddress(ipB.String())
				v4Flag = true
			} else if validIPv6(ipB.String()) {
				v6Address, err = subIntf.Ipv6.Addresses.NewAddress(ipB.String())
				v6Flag = true
			} else {
				log.Error("Invalid IP address " + ipB.String())
				continue
			}

			if err != nil {
				log.Error("Creation of address subtree failed!")
				return
			}
			if v4Flag {
				ygot.BuildEmptyTree(v4Address)

				ipStr := new(string)
				*ipStr = ipB.String()
				v4Address.Ip = ipStr
				v4Address.Config.Ip = ipStr
				v4Address.State.Ip = ipStr

				ipNetBNum, _ := ipNetB.Mask.Size()
				prfxLen := new(uint8)
				*prfxLen = uint8(ipNetBNum)
				v4Address.Config.PrefixLength = prfxLen
				v4Address.State.PrefixLength = prfxLen
			}
			if v6Flag {
				ygot.BuildEmptyTree(v6Address)

				ipStr := new(string)
				*ipStr = ipB.String()
				v6Address.Ip = ipStr
				v6Address.Config.Ip = ipStr
				v6Address.State.Ip = ipStr

				ipNetBNum, _ := ipNetB.Mask.Size()
				prfxLen := new(uint8)
				*prfxLen = uint8(ipNetBNum)
				v6Address.Config.PrefixLength = prfxLen
				v6Address.State.PrefixLength = prfxLen
			}
		}
	}
}

func (app *IntfApp) convertInternalToOCPortStatInfo(ifName *string, ifInfo *ocbinds.OpenconfigInterfaces_Interfaces_Interface) {
	if len(app.intfD.portStatMap) == 0 {
		log.Errorf("Port stat info not present for interface : %s", *ifName)
		return
	}
	if portStatInfo, ok := app.intfD.portStatMap[*ifName]; ok {

		inOctet := new(uint64)
		inOctetVal, _ := strconv.Atoi(portStatInfo.entry.Field["SAI_PORT_STAT_IF_IN_OCTETS"])
		*inOctet = uint64(inOctetVal)
		ifInfo.State.Counters.InOctets = inOctet

		inUCastPkt := new(uint64)
		inUCastPktVal, _ := strconv.Atoi(portStatInfo.entry.Field["SAI_PORT_STAT_IF_IN_UCAST_PKTS"])
		*inUCastPkt = uint64(inUCastPktVal)
		ifInfo.State.Counters.InUnicastPkts = inUCastPkt

		inNonUCastPkt := new(uint64)
		inNonUCastPktVal, _ := strconv.Atoi(portStatInfo.entry.Field["SAI_PORT_STAT_IF_IN_NON_UCAST_PKTS"])
		*inNonUCastPkt = uint64(inNonUCastPktVal)

		inPkt := new(uint64)
		*inPkt = *inUCastPkt + *inNonUCastPkt
		ifInfo.State.Counters.InPkts = inPkt

		inBCastPkt := new(uint64)
		inBCastPktVal, _ := strconv.Atoi(portStatInfo.entry.Field["SAI_PORT_STAT_IF_IN_BROADCAST_PKTS"])
		*inBCastPkt = uint64(inBCastPktVal)
		ifInfo.State.Counters.InBroadcastPkts = inBCastPkt

		inMCastPkt := new(uint64)
		inMCastPktVal, _ := strconv.Atoi(portStatInfo.entry.Field["SAI_PORT_STAT_IF_IN_MULTICAST_PKTS"])
		*inMCastPkt = uint64(inMCastPktVal)
		ifInfo.State.Counters.InMulticastPkts = inMCastPkt

		inErrPkt := new(uint64)
		inErrPktVal, _ := strconv.Atoi(portStatInfo.entry.Field["SAI_PORT_STAT_IF_IN_ERRORS"])
		*inErrPkt = uint64(inErrPktVal)
		ifInfo.State.Counters.InErrors = inErrPkt

		inDiscPkt := new(uint64)
		inDiscPktVal, _ := strconv.Atoi(portStatInfo.entry.Field["SAI_PORT_STAT_IF_IN_DISCARDS"])
		*inDiscPkt = uint64(inDiscPktVal)
		ifInfo.State.Counters.InDiscards = inDiscPkt

		outOctet := new(uint64)
		outOctetVal, _ := strconv.Atoi(portStatInfo.entry.Field["SAI_PORT_STAT_IF_OUT_OCTETS"])
		*outOctet = uint64(outOctetVal)
		ifInfo.State.Counters.OutOctets = outOctet

		outUCastPkt := new(uint64)
		outUCastPktVal, _ := strconv.Atoi(portStatInfo.entry.Field["SAI_PORT_STAT_IF_OUT_UCAST_PKTS"])
		*outUCastPkt = uint64(outUCastPktVal)
		ifInfo.State.Counters.OutUnicastPkts = outUCastPkt

		outNonUCastPkt := new(uint64)
		outNonUCastPktVal, _ := strconv.Atoi(portStatInfo.entry.Field["SAI_PORT_STAT_IF_OUT_NON_UCAST_PKTS"])
		*outNonUCastPkt = uint64(outNonUCastPktVal)

		outPkt := new(uint64)
		*outPkt = *outUCastPkt + *outNonUCastPkt
		ifInfo.State.Counters.OutPkts = outPkt

		outBCastPkt := new(uint64)
		outBCastPktVal, _ := strconv.Atoi(portStatInfo.entry.Field["SAI_PORT_STAT_IF_OUT_BROADCAST_PKTS"])
		*outBCastPkt = uint64(outBCastPktVal)
		ifInfo.State.Counters.OutBroadcastPkts = outBCastPkt

		outMCastPkt := new(uint64)
		outMCastPktVal, _ := strconv.Atoi(portStatInfo.entry.Field["SAI_PORT_STAT_IF_OUT_MULTICAST_PKTS"])
		*outMCastPkt = uint64(outMCastPktVal)
		ifInfo.State.Counters.OutMulticastPkts = outMCastPkt

		outErrPkt := new(uint64)
		outErrPktVal, _ := strconv.Atoi(portStatInfo.entry.Field["SAI_PORT_STAT_IF_OUT_ERRORS"])
		*outErrPkt = uint64(outErrPktVal)
		ifInfo.State.Counters.OutErrors = outErrPkt

		outDiscPkt := new(uint64)
		outDiscPktVal, _ := strconv.Atoi(portStatInfo.entry.Field["SAI_PORT_STAT_IF_OUT_DISCARDS"])
		*outDiscPkt = uint64(outDiscPktVal)
		ifInfo.State.Counters.OutDiscards = outDiscPkt
	}
}
