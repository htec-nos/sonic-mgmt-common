package translib

import (
	"fmt" // ADD THIS IMPORT
	"reflect"
	"strconv" // Better for string to int conversion
	"strings"

	"github.com/Azure/sonic-mgmt-common/cvl"
	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"

	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

const (
	BGP_GLOBALS_TABLE          = "BGP_GLOBALS"
	BGP_GLOBALS_AF_NETWORK_TAB = "BGP_GLOBALS_AF_NETWORK"
	BGP_NEIGHBOR_AF_TAB        = "BGP_NEIGHBOR_AF"
)

type BgpApp struct {
	pathInfo   *PathInfo
	ygotRoot   *ygot.GoStruct
	ygotTarget *interface{}

	bgpGlobalsTs  *db.TableSpec
	bgpGlobalsMap map[string]db.Value

	bgpGlobalsAfNetTs  *db.TableSpec
	bgpGlobalsAfNetMap map[string]db.Value

	bgpNeighborAfTs  *db.TableSpec
	bgpNeighborAfMap map[string]db.Value

	vrfName string
}

func init() {
	err := register("/openconfig-bgp:bgp",
		&appInfo{
			appType:       reflect.TypeOf(BgpApp{}),
			ygotRootType:  reflect.TypeOf(ocbinds.OpenconfigBgp_Bgp{}),
			isNative:      false,
			tablesToWatch: []*db.TableSpec{&db.TableSpec{Name: BGP_GLOBALS_TABLE}},
		})

	if err != nil {
		log.Fatal("Register BGP app module with App Interface failed with error=", err)
	}

	err = addModel(&ModelData{
		Name: "openconfig-bgp",
		Org:  "OpenConfig working group",
		Ver:  "9.0.0",
	})
	if err != nil {
		log.Fatal("Adding model data to appinterface failed with error=", err)
	}
}

func (app *BgpApp) initialize(data appData) {
	log.Info("initialize:bgp:path =", data.path)
	pathInfo := NewPathInfo(data.path)
	*app = BgpApp{
		pathInfo:   pathInfo,
		ygotRoot:   data.ygotRoot,
		ygotTarget: data.ygotTarget,
		vrfName:    "default",
	}

	app.bgpGlobalsTs = &db.TableSpec{Name: BGP_GLOBALS_TABLE}
	app.bgpGlobalsMap = make(map[string]db.Value)

	app.bgpGlobalsAfNetTs = &db.TableSpec{Name: BGP_GLOBALS_AF_NETWORK_TAB}
	app.bgpGlobalsAfNetMap = make(map[string]db.Value)

	app.bgpNeighborAfTs = &db.TableSpec{Name: BGP_NEIGHBOR_AF_TAB}
	app.bgpNeighborAfMap = make(map[string]db.Value)
}

func (app *BgpApp) getAppRootObject() *ocbinds.OpenconfigBgp_Bgp {
	deviceObj := (*app.ygotRoot).(*ocbinds.Device)
	if deviceObj.Bgp == nil {
		// Allocate BGP root so subsequent code can safely populate children
		deviceObj.Bgp = &ocbinds.OpenconfigBgp_Bgp{}
	}

	return deviceObj.Bgp
}

func (app *BgpApp) translateCreate(d *db.DB) ([]db.WatchKeys, error) {
	return app.translateCRUDCommon(d, CREATE)
}

func (app *BgpApp) translateReplace(d *db.DB) ([]db.WatchKeys, error) {
	return app.translateCRUDCommon(d, REPLACE)
}

func (app *BgpApp) translateUpdate(d *db.DB) ([]db.WatchKeys, error) {
	return app.translateCRUDCommon(d, UPDATE)
}

func (app *BgpApp) translateDelete(d *db.DB) ([]db.WatchKeys, error) {
	return app.translateCRUDCommon(d, DELETE)
}

func (app *BgpApp) translateGet(dbs [db.MaxDB]*db.DB) error {
	path := app.pathInfo.Template
	switch path {
	case "/openconfig-bgp:bgp/global":
		return app.translateGetBgpGlobals(dbs)
	case "/openconfig-bgp:bgp/global/afi-safis/afi-safi{}/openconfig-bgp-network-ext:networks":
		return app.translateGetBgpGlobalsAfNetwork(dbs)
	case "/openconfig-bgp:bgp/neighbors/neighbor{}/afi-safis/afi-safi{}":
		return app.translateGetBgpNeighborAf(dbs)
	default:
		return tlerr.NotSupported("Path not supported")
	}
}

func (app *BgpApp) translateAction(dbs [db.MaxDB]*db.DB) error {
	return tlerr.NotSupported("unsupported")
}

func (app *BgpApp) translateSubscribe(req translateSubRequest) (translateSubResponse, error) {
	return emptySubscribeResponse(req.path)
}

func (app *BgpApp) processCreate(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	if err = app.processCRUDCommon(d, CREATE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}
	return resp, err
}

func (app *BgpApp) processReplace(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	if err = app.processCRUDCommon(d, REPLACE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}
	return resp, err
}

func (app *BgpApp) processUpdate(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	if err = app.processCRUDCommon(d, UPDATE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}
	return resp, err
}

func (app *BgpApp) processDelete(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	if err = app.processCRUDCommon(d, DELETE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}
	return resp, err
}

func (app *BgpApp) processGet(dbs [db.MaxDB]*db.DB, fmtType TranslibFmtType) (GetResponse, error) {
	return generateGetResponse(app.pathInfo.Path, app.ygotRoot, fmtType)
}

func (app *BgpApp) processAction(dbs [db.MaxDB]*db.DB) (ActionResponse, error) {
	return ActionResponse{}, tlerr.New("not implemented")
}

func (app *BgpApp) processSubscribe(req processSubRequest) (processSubResponse, error) {
	return processSubResponse{}, tlerr.New("not implemented")
}

// =============================================================================
// Helper functions for CRUD common processing
// =============================================================================

func (app *BgpApp) translateCRUDCommon(configDB *db.DB, opcode int) ([]db.WatchKeys, error) {
	path := app.pathInfo.Template
	switch path {
	case "/openconfig-bgp:bgp/global", "/openconfig-bgp:bgp/global/config":
		return app.convertOCBgpGlobalsToInternal(opcode)
	case "/openconfig-bgp:bgp/global/afi-safis/afi-safi{}/openconfig-bgp-network-ext:networks/network{}",
		"/openconfig-bgp:bgp/global/afi-safis/afi-safi{}/openconfig-bgp-network-ext:networks/network{}/config":
		return app.convertOCBgpGlobalsAfNetworkToInternal(opcode)
	default:
		var keys []db.WatchKeys
		return keys, tlerr.NotSupported("Path not supported")
	}
}

func (app *BgpApp) processCRUDCommon(configDB *db.DB, opcode int) error {
	path := app.pathInfo.Template
	switch path {
	case "/openconfig-bgp:bgp/global", "/openconfig-bgp:bgp/global/config":
		return app.setBgpDataInConfigDb(configDB, app.bgpGlobalsTs, app.bgpGlobalsMap, opcode)
	case "/openconfig-bgp:bgp/global/afi-safis/afi-safi{}/openconfig-bgp-network-ext:networks/network{}",
		"/openconfig-bgp:bgp/global/afi-safis/afi-safi{}/openconfig-bgp-network-ext:networks/network{}/config":
		return app.setBgpDataInConfigDb(configDB, app.bgpGlobalsAfNetTs, app.bgpGlobalsAfNetMap, opcode)
	default:
		return tlerr.NotSupported("Path not supported")
	}
}

func (app *BgpApp) setBgpDataInConfigDb(configDB *db.DB, ts *db.TableSpec, dataMap map[string]db.Value, opcode int) error {
	for key, value := range dataMap {
		k := db.Key{Comp: []string{key}}
		existingEntry, getErr := configDB.GetEntry(ts, k)

		// Return error if GetEntry fails for reasons other than not found
		if getErr != nil && !isNotFoundError(getErr) {
			return getErr
		}

		var err error
		switch opcode {
		case CREATE:
			if existingEntry.IsPopulated() {
				return tlerr.AlreadyExists(fmt.Sprintf("%s entry '%s' already exists", ts.Name, key))
			}
			err = configDB.CreateEntry(ts, k, value)

		case REPLACE:
			if existingEntry.IsPopulated() {
				err = configDB.ModEntry(ts, k, value)
			} else {
				err = configDB.CreateEntry(ts, k, value)
			}

		case UPDATE:
			if !existingEntry.IsPopulated() {
				return tlerr.NotFound(fmt.Sprintf("%s entry '%s' not found", ts.Name, key))
			}
			err = configDB.ModEntry(ts, k, value)

		case DELETE:
			if !existingEntry.IsPopulated() {
				return tlerr.NotFound(fmt.Sprintf("%s entry '%v' not found", ts.Name, key))
			}

			cvlSess, cvlErr := configDB.NewValidationSession()
			if cvlErr != nil {
				return fmt.Errorf("failed to open CVL session: %v", cvlErr)
			}
			defer cvl.ValidationSessClose(cvlSess)

			redisKey := ts.Name + "|" + app.vrfName

			depEntries := cvlSess.GetDepDataForDelete(redisKey)

			// Delete dependent entries first
			for _, depData := range depEntries {
				for depKey := range depData.Entry {
					parts := strings.SplitN(depKey, "|", 2)
					if len(parts) == 2 {
						depTs := &db.TableSpec{Name: parts[0]}
						depKeyComps := strings.Split(parts[1], "|")
						configDB.DeleteEntry(depTs, db.Key{Comp: depKeyComps})
					}
				}
			}

			// Finally delete the parent entry
			err = configDB.DeleteEntry(ts, k)

		default:
			return fmt.Errorf("unsupported opcode %d", opcode)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// =============================================================================
// Helper functions for BGP_GLOBALS
// =============================================================================

// CRUD related

func (app *BgpApp) convertOCBgpGlobalsToInternal(opcode int) ([]db.WatchKeys, error) {
	var keys []db.WatchKeys

	if opcode == DELETE {
		// For DELETE, just populate the map with the VRF key
		// No need to read from YANG payload since DELETE has no payload
		app.bgpGlobalsMap = make(map[string]db.Value)
		app.bgpGlobalsMap[app.vrfName] = db.Value{Field: map[string]string{}}

		// Generate watch keys
		keys = append(keys, db.WatchKeys{
			Ts:  app.bgpGlobalsTs,
			Key: &db.Key{Comp: []string{app.vrfName}},
		})
		return keys, nil
	}

	bgp := app.getAppRootObject()
	if bgp != nil && bgp.Global != nil {
		app.bgpGlobalsMap[app.vrfName] = db.Value{Field: map[string]string{}}

		if bgp.Global.Config != nil {
			if bgp.Global.Config.As != nil {
				app.bgpGlobalsMap[app.vrfName].Field["local_asn"] = fmt.Sprint(*bgp.Global.Config.As)
			}
			if bgp.Global.Config.RouterId != nil {
				app.bgpGlobalsMap[app.vrfName].Field["router_id"] = *bgp.Global.Config.RouterId
			}
		}
		// Generate watch keys
		keys = append(keys, db.WatchKeys{
			Ts:  app.bgpGlobalsTs,
			Key: &db.Key{Comp: []string{app.vrfName}},
		})

		return keys, nil
	} else {
		return keys, tlerr.NotFound("BGP global configuration not found in YANG payload")
	}
}

// Get related

func (app *BgpApp) translateGetBgpGlobals(dbs [db.MaxDB]*db.DB) error {
	var err error
	bgp := app.getAppRootObject()
	configDB := dbs[db.ConfigDB]

	err = app.convertDBBgpGlobalsToInternal(configDB, db.Key{Comp: []string{app.vrfName}})
	if err != nil {
		return err
	}
	ygot.BuildEmptyTree(bgp.Global)
	app.convertInternalToOCBgpGlobals(bgp.Global)
	return nil
}

func (app *BgpApp) convertDBBgpGlobalsToInternal(configDB *db.DB, key db.Key) error {
	entry, err := configDB.GetEntry(app.bgpGlobalsTs, key)
	if err != nil {
		return err
	}
	if entry.IsPopulated() {
		app.bgpGlobalsMap[key.Get(0)] = entry
	} else {
		return tlerr.NotFound("BGP global configuration not found")
	}
	return nil
}

func (app *BgpApp) convertInternalToOCBgpGlobals(global *ocbinds.OpenconfigBgp_Bgp_Global) {
	if data, ok := app.bgpGlobalsMap[app.vrfName]; ok {
		// Ensure Config and State are initialized
		if global.Config == nil {
			global.Config = &ocbinds.OpenconfigBgp_Bgp_Global_Config{}
		}
		if global.State == nil {
			global.State = &ocbinds.OpenconfigBgp_Bgp_Global_State{}
		}

		if asn := data.Get("local_asn"); asn != "" {
			if asnVal, err := strconv.ParseUint(asn, 10, 32); err == nil {
				asnVal32 := uint32(asnVal)
				global.Config.As = &asnVal32
				global.State.As = &asnVal32
			} else {
				log.Errorf("Failed to parse ASN: %v", err)
			}
		}
		if routerId := data.Get("router_id"); routerId != "" {
			global.Config.RouterId = &routerId
			global.State.RouterId = &routerId
		}
	}
}

// =============================================================================
// Helper functions for BGP_GLOBALS_AF_NETWORK
// =============================================================================

// CRUD related

func (app *BgpApp) convertOCBgpGlobalsAfNetworkToInternal(opcode int) ([]db.WatchKeys, error) {
	var keys []db.WatchKeys
	afiSafi := strings.ToLower(app.pathInfo.Var("afi-safi-name"))
	prefix := app.pathInfo.Var("prefix")

	dbKey := fmt.Sprintf("%s|%s|%s", app.vrfName, afiSafi, prefix)

	if opcode == DELETE {
		// For DELETE, just populate the map with the VRF, AFI-SAFI and prefix key
		// No need to read from YANG payload since DELETE has no payload
		app.bgpGlobalsAfNetMap = make(map[string]db.Value)

		app.bgpGlobalsAfNetMap[dbKey] = db.Value{Field: map[string]string{}}

		keys = append(keys, db.WatchKeys{
			Ts:  app.bgpGlobalsAfNetTs,
			Key: &db.Key{Comp: []string{app.vrfName, afiSafi, prefix}},
		})
		return keys, nil
	}

	bgp := app.getAppRootObject()
	if bgp == nil || bgp.Global == nil || bgp.Global.AfiSafis == nil {
		return keys, tlerr.NotFound("BGP AFI-SAFI networks configuration not found in YANG payload")
	}

	// Parse the AFI-SAFI enum
	afiSafiEnum, err := parseAfiSafiType(afiSafi)
	if err != nil {
		return keys, err
	}

	// Get the specific AFI-SAFI entry
	afiSafiEntry, exists := bgp.Global.AfiSafis.AfiSafi[afiSafiEnum]
	if !exists || afiSafiEntry.Networks == nil {
		return keys, tlerr.NotFound("AFI-SAFI networks not found in YANG payload")
	}

	// Find the specific network entry by prefix
	networkEntry, exists := afiSafiEntry.Networks.Network[prefix]
	if !exists {
		return keys, tlerr.NotFound("Network prefix not found in YANG payload")
	}

	// Initialize the map entry
	app.bgpGlobalsAfNetMap = make(map[string]db.Value)
	app.bgpGlobalsAfNetMap[dbKey] = db.Value{Field: map[string]string{}}

	// Extract config fields if present
	hasFields := false
	if networkEntry.Config != nil {
		if networkEntry.Config.Prefix != nil {
			// Prefix is part of the key, so we don't store it as a field
			// But we validate it matches
			if *networkEntry.Config.Prefix != prefix {
				return keys, tlerr.InvalidArgs("Config prefix does not match path prefix")
			}
		}
		if networkEntry.Config.Policy != nil && *networkEntry.Config.Policy != "" {
			app.bgpGlobalsAfNetMap[dbKey].Field["policy"] = *networkEntry.Config.Policy
			hasFields = true
		}
		if networkEntry.Config.Backdoor != nil {
			backdoorStr := fmt.Sprint(*networkEntry.Config.Backdoor)
			if backdoorStr != "" {
				app.bgpGlobalsAfNetMap[dbKey].Field["backdoor"] = backdoorStr
				hasFields = true
			}
		}
	}

	if !hasFields {
		app.bgpGlobalsAfNetMap[dbKey].Field["NULL"] = "NULL"
	}

	// Generate watch keys
	keys = append(keys, db.WatchKeys{
		Ts:  app.bgpGlobalsAfNetTs,
		Key: &db.Key{Comp: []string{app.vrfName, afiSafi, prefix}},
	})

	return keys, nil
}

// Get related

func (app *BgpApp) translateGetBgpGlobalsAfNetwork(dbs [db.MaxDB]*db.DB) error {
	configDB := dbs[db.ConfigDB]
	afiSafi := strings.ToLower(app.pathInfo.Var("afi-safi-name"))

	err := app.convertDBBgpGlobalsAfNetworkToInternal(configDB, afiSafi)
	if err != nil {
		return err
	}

	bgp := app.getAppRootObject()
	ygot.BuildEmptyTree(bgp.Global)
	app.convertInternalToOCBgpAfNetwork(afiSafi, bgp.Global)
	return nil
}

func (app *BgpApp) convertDBBgpGlobalsAfNetworkToInternal(configDB *db.DB, afiSafi string) error {
	app.bgpGlobalsAfNetMap = make(map[string]db.Value)

	entries, err := configDB.GetKeys(app.bgpGlobalsAfNetTs)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return tlerr.NotFound("BGP_GLOBALS_AF_NETWORK configuration not found")
	}

	for _, k := range entries {
		if len(k.Comp) < 3 {
			continue
		}
		if k.Comp[0] == app.vrfName && k.Comp[1] == afiSafi {
			val, _ := configDB.GetEntry(app.bgpGlobalsAfNetTs, k)
			app.bgpGlobalsAfNetMap[strings.Join(k.Comp, "|")] = val
		}
	}
	return nil
}

func (app *BgpApp) convertInternalToOCBgpAfNetwork(afiSafi string, global *ocbinds.OpenconfigBgp_Bgp_Global) {
	if global.AfiSafis == nil {
		global.AfiSafis = &ocbinds.OpenconfigBgp_Bgp_Global_AfiSafis{}
	}

	// Parse the afiSafi string to enum
	afiSafiEnum, err := parseAfiSafiType(afiSafi)
	if err != nil {
		log.Errorf("Failed to parse AFI-SAFI type %s: %v", afiSafi, err)
		return
	}

	// Initialize the map if needed
	if global.AfiSafis.AfiSafi == nil {
		global.AfiSafis.AfiSafi = make(map[ocbinds.E_OpenconfigBgpTypes_AFI_SAFI_TYPE]*ocbinds.OpenconfigBgp_Bgp_Global_AfiSafis_AfiSafi)
	}

	// Get or create the AFI-SAFI entry
	afi, exists := global.AfiSafis.AfiSafi[afiSafiEnum]
	if !exists {
		afi = &ocbinds.OpenconfigBgp_Bgp_Global_AfiSafis_AfiSafi{}
		global.AfiSafis.AfiSafi[afiSafiEnum] = afi
	}

	// Initialize Networks container
	if afi.Networks == nil {
		afi.Networks = &ocbinds.OpenconfigBgp_Bgp_Global_AfiSafis_AfiSafi_Networks{}
	}

	// Iterate over internal map and populate OC structures
	for keyStr, v := range app.bgpGlobalsAfNetMap {

		parts := strings.Split(keyStr, "|")
		if len(parts) < 3 {
			continue
		}
		prefix := parts[2]
		policy := v.Get("policy")
		backdoor := v.Get("backdoor") == "true"

		net := &ocbinds.OpenconfigBgp_Bgp_Global_AfiSafis_AfiSafi_Networks_Network{
			Prefix: ygot.String(prefix),
			Config: &ocbinds.OpenconfigBgp_Bgp_Global_AfiSafis_AfiSafi_Networks_Network_Config{
				Prefix:   ygot.String(prefix),
				Policy:   ygot.String(policy),
				Backdoor: ygot.Bool(backdoor),
			},
			State: &ocbinds.OpenconfigBgp_Bgp_Global_AfiSafis_AfiSafi_Networks_Network_State{
				Prefix:   ygot.String(prefix),
				Policy:   ygot.String(policy),
				Backdoor: ygot.Bool(backdoor),
			},
		}

		// Add to Networks
		if afi.Networks.Network == nil {
			afi.Networks.Network = make(map[string]*ocbinds.OpenconfigBgp_Bgp_Global_AfiSafis_AfiSafi_Networks_Network)
		}
		afi.Networks.Network[prefix] = net
	}
}

// Helper function to parse AFI-SAFI string to enum - returns error if invalid
func parseAfiSafiType(afiSafi string) (ocbinds.E_OpenconfigBgpTypes_AFI_SAFI_TYPE, error) {
	switch afiSafi {
	case "ipv4_unicast", "IPV4_UNICAST":
		return ocbinds.OpenconfigBgpTypes_AFI_SAFI_TYPE_IPV4_UNICAST, nil
	case "ipv6_unicast", "IPV6_UNICAST":
		return ocbinds.OpenconfigBgpTypes_AFI_SAFI_TYPE_IPV6_UNICAST, nil
	case "l2vpn_evpn", "L2VPN_EVPN":
		return ocbinds.OpenconfigBgpTypes_AFI_SAFI_TYPE_L2VPN_EVPN, nil
	default:
		return ocbinds.OpenconfigBgpTypes_AFI_SAFI_TYPE_UNSET,
			fmt.Errorf("unsupported AFI-SAFI type: %s", afiSafi)
	}
}

// =============================================================================
// Helper functions for BGP_NEIGHBOR_AF
// =============================================================================

// Get related

func (app *BgpApp) translateGetBgpNeighborAf(dbs [db.MaxDB]*db.DB) error {

	configDB := dbs[db.ConfigDB]
	vrfName := "default"
	// Extract neighbor IP and AFI-SAFI from the request path
	neighbor := app.pathInfo.Var("neighbor-address")
	afiSafi := strings.ToLower(app.pathInfo.Var("afi-safi-name"))

	// Read entries from ConfigDB
	err := app.convertDBBgpNeighborAfToInternal(configDB, vrfName, neighbor, afiSafi)
	if err != nil {
		return err
	}

	// Convert internal map to YANG (OpenConfig)
	bgp := app.getAppRootObject()
	// Ensure Neighbors container exists
	if bgp.Neighbors == nil {
		bgp.Neighbors = &ocbinds.OpenconfigBgp_Bgp_Neighbors{}
	}

	// Build the AF subtree for this neighbor
	app.convertInternalToOCBgpNeighborAf(vrfName, neighbor, afiSafi, bgp.Neighbors)

	return nil
}

/*
 * Reads a single BGP_NEIGHBOR_AF entry from CONFIG_DB for a specific
 * neighbor and AFI-SAFI, and stores it in the internal map (bgpNeighborAfMap).
 *
 * Key format in DB: <vrf>|<neighbor>|<afi-safi>
 */
func (app *BgpApp) convertDBBgpNeighborAfToInternal(configDB *db.DB, vrfName, neighbor, afiSafi string) error {

	// Initialize the internal map to store BGP neighbor AF entries
	app.bgpNeighborAfMap = make(map[string]db.Value)

	// Construct the DB key for the neighbor AFI-SAFI
	key := fmt.Sprintf("%s|%s|%s", vrfName, neighbor, afiSafi)
	dbKey := db.Key{Comp: strings.Split(key, "|")}

	// Read entry from CONFIG_DB
	entry, err := configDB.GetEntry(app.bgpNeighborAfTs, dbKey)
	if err != nil {
		log.Errorf("Failed to read CONFIG_DB entry for BGP_NEIGHBOR_AF %s: %v", key, err)
		return err
	}

	if !entry.IsPopulated() {
		return tlerr.NotFound(
			fmt.Sprintf("BGP_NEIGHBOR_AF entry not found for %s|%s|%s",
				vrfName, neighbor, afiSafi))
	}

	// Store the DB value in the internal map
	app.bgpNeighborAfMap[key] = entry

	// Print the entry contents
	log.Infof("BGP_NEIGHBOR_AF internal map entry for %s:", key)
	for field, value := range entry.Field {
		log.Infof("   %s : %s", field, value)
	}

	log.Infof("Finished reading BGP_NEIGHBOR_AF entry for %s", key)

	return nil
}

func (app *BgpApp) convertInternalToOCBgpNeighborAf(
	vrfName, neighbor, afiSafi string,
	neighbors *ocbinds.OpenconfigBgp_Bgp_Neighbors,
) {
	// Ensure neighbors container exists
	if neighbors.Neighbor == nil {
		neighbors.Neighbor = make(map[string]*ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor)
	}

	// Get or allocate neighbor entry
	nb, ok := neighbors.Neighbor[neighbor]
	if !ok {
		nb = &ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor{
			NeighborAddress: ygot.String(neighbor),
		}
		neighbors.Neighbor[neighbor] = nb
	}

	// Ensure afi-safi container exists
	if nb.AfiSafis == nil {
		nb.AfiSafis = &ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor_AfiSafis{}
	}
	if nb.AfiSafis.AfiSafi == nil {
		nb.AfiSafis.AfiSafi = make(
			map[ocbinds.E_OpenconfigBgpTypes_AFI_SAFI_TYPE]*ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor_AfiSafis_AfiSafi)
	}

	// Look up internal model for this Neighbor AF
	key := fmt.Sprintf("%s|%s|%s", vrfName, neighbor, afiSafi)
	afInternal, ok := app.bgpNeighborAfMap[key]
	if !ok {
		// nothing to populate
		return
	}

	// Convert afiSafi to OpenConfig enum
	ocAfi, err := parseAfiSafiType(afiSafi)
	if err != nil {
		// unknown AFI-SAFI
		log.Warningf("Unsupported afi-safi %s: %v", afiSafi, err)
		return
	}

	// Create OC afi-safi entry
	afEntry, exists := nb.AfiSafis.AfiSafi[ocAfi]
	if !exists {
		afEntry = &ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor_AfiSafis_AfiSafi{
			AfiSafiName: ocAfi,
			Config:      &ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor_AfiSafis_AfiSafi_Config{},
			State:       &ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor_AfiSafis_AfiSafi_State{},
		}
		nb.AfiSafis.AfiSafi[ocAfi] = afEntry
	}

	// Ensure Config and State are initialized
	if afEntry.Config == nil {
		afEntry.Config = &ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor_AfiSafis_AfiSafi_Config{}
	}
	if afEntry.State == nil {
		afEntry.State = &ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor_AfiSafis_AfiSafi_State{}
	}

	cfg := afEntry.Config
	st := afEntry.State

	// Populate Enabled
	val := afInternal.Get("admin_status")
	if val != "" {
		b := val == "true"
		cfg.Enabled = ygot.Bool(b)
		st.Enabled = ygot.Bool(b)
	}

	//Allocate neighbor ApplyPolicy if needed
	if afEntry.ApplyPolicy == nil {
		afEntry.ApplyPolicy = &ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor_AfiSafis_AfiSafi_ApplyPolicy{
			Config: &ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor_AfiSafis_AfiSafi_ApplyPolicy_Config{},
			State:  &ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor_AfiSafis_AfiSafi_ApplyPolicy_State{},
		}
	}

	apc := afEntry.ApplyPolicy.Config
	aps := afEntry.ApplyPolicy.State

	// Populate policies
	if val := afInternal.GetList("route_map_in"); len(val) > 0 {
		apc.ImportPolicy = val
		aps.ImportPolicy = val
	}

	if val := afInternal.GetList("route_map_out"); len(val) > 0 {
		apc.ExportPolicy = val
		aps.ExportPolicy = val
	}

	// Allocate neighbor-afi-safi-ext extension if needed
	if afEntry.NeighborAfiSafiExt == nil {
		afEntry.NeighborAfiSafiExt = &ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor_AfiSafis_AfiSafi_NeighborAfiSafiExt{
			Config: &ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor_AfiSafis_AfiSafi_NeighborAfiSafiExt_Config{},
			State:  &ocbinds.OpenconfigBgp_Bgp_Neighbors_Neighbor_AfiSafis_AfiSafi_NeighborAfiSafiExt_State{},
		}
	}

	extCfg := afEntry.NeighborAfiSafiExt.Config
	extState := afEntry.NeighborAfiSafiExt.State

	// Map nhself (next-hop-self)
	if val := afInternal.Get("nhself"); val != "" {
		b := val == "true"
		extCfg.NextHopSelf = ygot.Bool(b)
		extState.NextHopSelf = ygot.Bool(b)
	}

	// Map rrclient (route-reflector-client)
	if val := afInternal.Get("rrclient"); val != "" {
		b := val == "true"
		extCfg.RouteReflectorClient = ygot.Bool(b)
		extState.RouteReflectorClient = ygot.Bool(b)
	}
}
