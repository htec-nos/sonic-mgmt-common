package translib

import (
	"fmt" // ADD THIS IMPORT
	"reflect"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"

	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

const (
	ROUTE_MAP_TABLE  = "ROUTE_MAP"
	PREFIX_SET_TABLE = "PREFIX_SET"
	PREFIX_TABLE     = "PREFIX"
)

type RoutingPolicyApp struct {
	pathInfo   *PathInfo
	ygotRoot   *ygot.GoStruct
	ygotTarget *interface{}

	routeMapTs  *db.TableSpec
	routeMapMap map[string]db.Value

	prefixSetTs  *db.TableSpec
	prefixSetMap map[string]db.Value

	prefixTs  *db.TableSpec
	prefixMap map[string]db.Value
}

func init() {
	err := register("/openconfig-routing-policy:routing-policy",
		&appInfo{
			appType:       reflect.TypeOf(RoutingPolicyApp{}),
			ygotRootType:  reflect.TypeOf(ocbinds.OpenconfigRoutingPolicy_RoutingPolicy{}),
			isNative:      false,
			tablesToWatch: []*db.TableSpec{&db.TableSpec{Name: ROUTE_MAP_TABLE}},
		})

	if err != nil {
		log.Fatal("Register BGP app module with App Interface failed with error=", err)
	}

	err = addModel(&ModelData{
		Name: "openconfig-routing-policy",
		Org:  "OpenConfig working group",
		Ver:  "9.0.0",
	})
	if err != nil {
		log.Fatal("Adding model data to appinterface failed with error=", err)
	}
}

func (app *RoutingPolicyApp) initialize(data appData) {
	log.Info("initialize:routing-policy:path =", data.path)
	pathInfo := NewPathInfo(data.path)
	*app = RoutingPolicyApp{
		pathInfo:   pathInfo,
		ygotRoot:   data.ygotRoot,
		ygotTarget: data.ygotTarget,
	}

	app.routeMapTs = &db.TableSpec{Name: ROUTE_MAP_TABLE}
	app.routeMapMap = make(map[string]db.Value)

	app.prefixSetTs = &db.TableSpec{Name: PREFIX_SET_TABLE}
	app.prefixSetMap = make(map[string]db.Value)

	app.prefixTs = &db.TableSpec{Name: PREFIX_TABLE}
	app.prefixMap = make(map[string]db.Value)
}

func (app *RoutingPolicyApp) getAppRootObject() *ocbinds.OpenconfigRoutingPolicy_RoutingPolicy {
	deviceObj := (*app.ygotRoot).(*ocbinds.Device)
	return deviceObj.RoutingPolicy
}

func (app *RoutingPolicyApp) translateCreate(d *db.DB) ([]db.WatchKeys, error) {
	return app.translateCRUDCommon(d, CREATE)
}

func (app *RoutingPolicyApp) translateReplace(d *db.DB) ([]db.WatchKeys, error) {
	return app.translateCRUDCommon(d, REPLACE)
}

func (app *RoutingPolicyApp) translateUpdate(d *db.DB) ([]db.WatchKeys, error) {
	return app.translateCRUDCommon(d, UPDATE)
}

func (app *RoutingPolicyApp) translateDelete(d *db.DB) ([]db.WatchKeys, error) {
	return app.translateCRUDCommon(d, DELETE)
}

func (app *RoutingPolicyApp) translateGet(dbs [db.MaxDB]*db.DB) error {
	path := app.pathInfo.Template
	switch path {
	case "/openconfig-routing-policy:routing-policy/policy-definitions":
		return app.translateGetRouteMap(dbs)
	case "/openconfig-routing-policy:routing-policy/defined-sets/prefix-sets":
		return app.translateGetPrefixSet(dbs)
	default:
		return tlerr.NotSupported("Path not supported")
	}
}

func (app *RoutingPolicyApp) translateAction(dbs [db.MaxDB]*db.DB) error {
	return tlerr.NotSupported("unsupported")
}

func (app *RoutingPolicyApp) translateSubscribe(req translateSubRequest) (translateSubResponse, error) {
	return emptySubscribeResponse(req.path)
}

func (app *RoutingPolicyApp) processCreate(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	if err = app.processCRUDCommon(d, CREATE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}
	return resp, err
}

func (app *RoutingPolicyApp) processReplace(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	if err = app.processCRUDCommon(d, REPLACE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}
	return resp, err
}

func (app *RoutingPolicyApp) processUpdate(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	if err = app.processCRUDCommon(d, UPDATE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}
	return resp, err
}

func (app *RoutingPolicyApp) processDelete(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	if err = app.processCRUDCommon(d, DELETE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}
	return resp, err
}

func (app *RoutingPolicyApp) processGet(dbs [db.MaxDB]*db.DB, fmtType TranslibFmtType) (GetResponse, error) {
	return generateGetResponse(app.pathInfo.Path, app.ygotRoot, fmtType)
}

func (app *RoutingPolicyApp) processAction(dbs [db.MaxDB]*db.DB) (ActionResponse, error) {
	return ActionResponse{}, tlerr.New("not implemented")
}

func (app *RoutingPolicyApp) processSubscribe(req processSubRequest) (processSubResponse, error) {
	return processSubResponse{}, tlerr.New("not implemented")
}

// =============================================================================
// Helper functions for CRUD common processing
// =============================================================================

func (app *RoutingPolicyApp) translateCRUDCommon(configDB *db.DB, opcode int) ([]db.WatchKeys, error) {
	path := app.pathInfo.Template
	switch path {
	case "/openconfig-routing-policy:routing-policy/policy-definitions/policy-definition{}/statements/statement{}":
		return app.convertOCRouteMapToInternal(opcode)
	default:
		var keys []db.WatchKeys
		return keys, tlerr.NotSupported("Path not supported")
	}
}

func (app *RoutingPolicyApp) processCRUDCommon(configDB *db.DB, opcode int) error {
	path := app.pathInfo.Template
	switch path {
	case "/openconfig-routing-policy:routing-policy/policy-definitions/policy-definition{}/statements/statement{}":
		return app.setDataInConfigDb(configDB, app.routeMapTs, app.routeMapMap, opcode)
	default:
		return tlerr.NotSupported("Path not supported")
	}
	return tlerr.NotSupported("Path not supported")
}

func (app *RoutingPolicyApp) setDataInConfigDb(configDB *db.DB, ts *db.TableSpec, dataMap map[string]db.Value, opcode int) error {
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
// Helper functions for ROUTE_MAP
// =============================================================================

// CRUD related

func (app *RoutingPolicyApp) convertOCRouteMapToInternal(opcode int) ([]db.WatchKeys, error) {
	var keys []db.WatchKeys

	polName := app.pathInfo.Var("name")
	stmtName := app.pathInfo.Var("name#2")

	dbKey := fmt.Sprintf("%s|%s", polName, stmtName)

	if opcode == DELETE {
		// For DELETE, just populate the map with the VRF key
		// No need to read from YANG payload since DELETE has no payload
		app.routeMapMap = make(map[string]db.Value)
		app.routeMapMap[dbKey] = db.Value{Field: map[string]string{}}

		// Generate watch keys
		keys = append(keys, db.WatchKeys{
			Ts:  app.routeMapTs,
			Key: &db.Key{Comp: strings.Split(dbKey, "|")},
		})
		return keys, nil
	}

	rp := app.getAppRootObject()

	if rp.PolicyDefinitions == nil || rp.PolicyDefinitions.PolicyDefinition == nil {
		return keys, tlerr.NotFound("Policy definitions not found in payload")
	}

	polDef, exists := rp.PolicyDefinitions.PolicyDefinition[polName]
	if !exists {
		return keys, tlerr.NotFound(fmt.Sprintf("Policy definition '%s' not found", polName))
	}

	if polDef.Statements == nil || polDef.Statements.Statement == nil {
		return keys, tlerr.NotFound(fmt.Sprintf("Statements not found for policy '%s'", polName))
	}

	stmt, exists := polDef.Statements.Statement[stmtName]
	if !exists {
		return keys, tlerr.NotFound(fmt.Sprintf("Statement '%s' not found in policy '%s'", stmtName, polName))
	}

	// Initialize the map
	app.routeMapMap = make(map[string]db.Value)
	// Create the db.Value for this route map entry
	routeMapData := db.Value{Field: make(map[string]string)}

	// Map Actions to DB fields
	if stmt.Actions != nil && stmt.Actions.Config != nil {
		if stmt.Actions.Config.PolicyResult == ocbinds.OpenconfigRoutingPolicy_PolicyResultType_ACCEPT_ROUTE {
			routeMapData.Field["route_operation"] = "permit"
		} else if stmt.Actions.Config.PolicyResult == ocbinds.OpenconfigRoutingPolicy_PolicyResultType_REJECT_ROUTE {
			routeMapData.Field["route_operation"] = "deny"
		}
	}

	// Map Conditions to DB fields
	if stmt.Conditions != nil {
		// Match prefix set
		if stmt.Conditions.MatchPrefixSet != nil && stmt.Conditions.MatchPrefixSet.Config != nil {
			if stmt.Conditions.MatchPrefixSet.Config.PrefixSet != nil {
				routeMapData.Field["match_prefix_set"] = *stmt.Conditions.MatchPrefixSet.Config.PrefixSet
			}
		}
	}

	// Store in the map
	app.routeMapMap[dbKey] = routeMapData

	// Generate watch keys
	keys = append(keys, db.WatchKeys{
		Ts:  app.routeMapTs,
		Key: &db.Key{Comp: strings.Split(dbKey, "|")},
	})

	return keys, nil
}

// Get related

func (app *RoutingPolicyApp) translateGetRouteMap(dbs [db.MaxDB]*db.DB) error {
	configDB := dbs[db.ConfigDB]

	err := app.convertDBRouteMapToInternal(configDB)
	if err != nil {
		return err
	}

	rp := app.getAppRootObject()
	ygot.BuildEmptyTree(rp.PolicyDefinitions)
	app.convertInternalToOCRouteMap(rp.PolicyDefinitions)
	return nil
}

func (app *RoutingPolicyApp) convertDBRouteMapToInternal(configDB *db.DB) error {
	app.routeMapMap = make(map[string]db.Value)

	entries, err := configDB.GetKeys(app.routeMapTs)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return tlerr.NotFound("ROUTE_MAP configuration not found")
	}

	for _, k := range entries {
		val, _ := configDB.GetEntry(app.routeMapTs, k)
		app.routeMapMap[strings.Join(k.Comp, "|")] = val
	}
	return nil
}

func (app *RoutingPolicyApp) convertInternalToOCRouteMap(policyDefs *ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions) {
	if policyDefs.PolicyDefinition == nil {
		policyDefs.PolicyDefinition = make(map[string]*ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition)
	}

	// Iterate over internal map and populate OC structures directly
	for keyStr, v := range app.routeMapMap {
		parts := strings.Split(keyStr, "|")
		if len(parts) < 2 {
			continue
		}
		polName := parts[0]
		seqNum := parts[1]

		// Get or create the policy definition
		polDef, exists := policyDefs.PolicyDefinition[polName]
		if !exists {
			polDef = &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition{
				Name: ygot.String(polName),
				Config: &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition_Config{
					Name: ygot.String(polName),
				},
			}
			policyDefs.PolicyDefinition[polName] = polDef
		}

		// Initialize Statements container
		if polDef.Statements == nil {
			polDef.Statements = &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition_Statements{}
		}
		if polDef.Statements.Statement == nil {
			polDef.Statements.Statement = make(map[string]*ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition_Statements_Statement)
		}

		// Get field values from Redis
		routeOp := v.Get("route_operation")
		matchPrefixSet := v.Get("match_prefix_set")

		// Create statement with config and state
		stmt := &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition_Statements_Statement{
			Name: ygot.String(seqNum),
			Config: &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition_Statements_Statement_Config{
				Name: ygot.String(seqNum),
			},
			State: &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition_Statements_Statement_State{
				Name: ygot.String(seqNum),
			},
		}

		// Initialize Conditions container
		stmt.Conditions = &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition_Statements_Statement_Conditions{
			Config: &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition_Statements_Statement_Conditions_Config{},
		}

		// Map match_prefix_set to conditions
		if matchPrefixSet != "" {
			stmt.Conditions.MatchPrefixSet = &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition_Statements_Statement_Conditions_MatchPrefixSet{
				Config: &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition_Statements_Statement_Conditions_MatchPrefixSet_Config{
					PrefixSet: ygot.String(matchPrefixSet),
				},
				State: &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition_Statements_Statement_Conditions_MatchPrefixSet_State{
					PrefixSet: ygot.String(matchPrefixSet),
				},
			}
		}

		// Initialize Actions container
		stmt.Actions = &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition_Statements_Statement_Actions{
			Config: &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition_Statements_Statement_Actions_Config{},
			State:  &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_PolicyDefinitions_PolicyDefinition_Statements_Statement_Actions_State{},
		}

		// Map route_operation to policy result in both config and state
		if routeOp == "permit" {
			stmt.Actions.Config.PolicyResult = ocbinds.OpenconfigRoutingPolicy_PolicyResultType_ACCEPT_ROUTE
			stmt.Actions.State.PolicyResult = ocbinds.OpenconfigRoutingPolicy_PolicyResultType_ACCEPT_ROUTE
		} else if routeOp == "deny" {
			stmt.Actions.Config.PolicyResult = ocbinds.OpenconfigRoutingPolicy_PolicyResultType_REJECT_ROUTE
			stmt.Actions.State.PolicyResult = ocbinds.OpenconfigRoutingPolicy_PolicyResultType_REJECT_ROUTE
		}

		// Add statement to policy definition
		polDef.Statements.Statement[seqNum] = stmt
	}
}

// =============================================================================
// Helper functions for PREFIX_SET
// =============================================================================

// Get related

func (app *RoutingPolicyApp) translateGetPrefixSet(dbs [db.MaxDB]*db.DB) error {
	configDB := dbs[db.ConfigDB]

	err := app.convertDBPrefixSetToInternal(configDB)
	if err != nil {
		return err
	}

	rp := app.getAppRootObject()
	ygot.BuildEmptyTree(rp.DefinedSets)
	app.convertInternalToOCPrefixSet(rp.DefinedSets)
	return nil
}

func (app *RoutingPolicyApp) convertDBPrefixSetToInternal(configDB *db.DB) error {
	app.prefixSetMap = make(map[string]db.Value)

	// Get PREFIX_SET entries
	entries, err := configDB.GetKeys(app.prefixSetTs)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return tlerr.NotFound("PREFIX_SET configuration not found")
	}

	for _, k := range entries {
		val, _ := configDB.GetEntry(app.prefixSetTs, k)
		app.prefixSetMap[strings.Join(k.Comp, "|")] = val
	}

	// Get PREFIX entries
	entries, err = configDB.GetKeys(app.prefixTs)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return tlerr.NotFound("PREFIX_SET configuration not found")
	}

	for _, k := range entries {
		val, _ := configDB.GetEntry(app.prefixTs, k)
		app.prefixMap[strings.Join(k.Comp, "|")] = val
	}

	return nil
}

func (app *RoutingPolicyApp) convertInternalToOCPrefixSet(definedSets *ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets) {
	// Initialize PrefixSets container if needed
	if definedSets.PrefixSets == nil {
		definedSets.PrefixSets = &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets{}
	}
	if definedSets.PrefixSets.PrefixSet == nil {
		definedSets.PrefixSets.PrefixSet = make(map[string]*ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets_PrefixSet)
	}

	// First, create all prefix sets from PREFIX_SET table
	for prefixSetName, v := range app.prefixSetMap {
		prefixSet := &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets_PrefixSet{
			Name: ygot.String(prefixSetName),
			Config: &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets_PrefixSet_Config{
				Name: ygot.String(prefixSetName),
			},
			State: &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets_PrefixSet_State{
				Name: ygot.String(prefixSetName),
			},
		}

		// Get mode from PREFIX_SET entry
		mode := v.Get("mode")
		if mode != "" {
			// Map mode to OpenConfig enum
			// "ipv4" -> IPV4, "ipv6" -> IPV6
			var modeEnum ocbinds.E_OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets_PrefixSet_Config_Mode
			if mode == "ipv4" {
				modeEnum = ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets_PrefixSet_Config_Mode_IPV4
			} else if mode == "ipv6" {
				modeEnum = ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets_PrefixSet_Config_Mode_IPV6
			}

			// Set mode in both config and state
			if modeEnum != 0 {
				prefixSet.Config.Mode = modeEnum
				prefixSet.State.Mode = modeEnum
			}
		}
		// Initialize Prefixes container
		prefixSet.Prefixes = &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets_PrefixSet_Prefixes{}
		prefixSet.Prefixes.Prefix = make(map[ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets_PrefixSet_Prefixes_Prefix_Key]*ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets_PrefixSet_Prefixes_Prefix)

		definedSets.PrefixSets.PrefixSet[prefixSetName] = prefixSet
	}

	// Now process PREFIX entries - only for prefix sets that exist in PREFIX_SET
	for keyStr := range app.prefixMap {
		parts := strings.Split(keyStr, "|")
		if len(parts) < 3 {
			continue
		}
		prefixSetName := parts[0]
		ipPrefix := parts[1]
		masklengthRange := parts[2]

		// Only process if this prefix belongs to a defined prefix set
		prefixSet, exists := definedSets.PrefixSets.PrefixSet[prefixSetName]
		if !exists {
			// Skip prefixes that don't belong to any defined prefix set
			continue
		}

		// Create prefix key
		prefixKey := ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets_PrefixSet_Prefixes_Prefix_Key{
			IpPrefix:        ipPrefix,
			MasklengthRange: masklengthRange,
		}

		// Create prefix entry
		prefix := &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets_PrefixSet_Prefixes_Prefix{
			IpPrefix:        ygot.String(ipPrefix),
			MasklengthRange: ygot.String(masklengthRange),
			Config: &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets_PrefixSet_Prefixes_Prefix_Config{
				IpPrefix:        ygot.String(ipPrefix),
				MasklengthRange: ygot.String(masklengthRange),
			},
			State: &ocbinds.OpenconfigRoutingPolicy_RoutingPolicy_DefinedSets_PrefixSets_PrefixSet_Prefixes_Prefix_State{
				IpPrefix:        ygot.String(ipPrefix),
				MasklengthRange: ygot.String(masklengthRange),
			},
		}

		// Add prefix to the prefix set
		prefixSet.Prefixes.Prefix[prefixKey] = prefix
	}
}
