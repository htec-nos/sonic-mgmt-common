package translib

import (
	"fmt" // ADD THIS IMPORT
	"reflect"
	"strconv" // Better for string to int conversion

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"

	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

const (
	BGP_GLOBALS_TABLE = "BGP_GLOBALS"
)

type BgpApp struct {
	pathInfo   *PathInfo
	ygotRoot   *ygot.GoStruct
	ygotTarget *interface{}

	bgpGlobalsTs  *db.TableSpec
	bgpGlobalsMap map[string]db.Value
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
	}

	app.bgpGlobalsTs = &db.TableSpec{Name: BGP_GLOBALS_TABLE}
	app.bgpGlobalsMap = make(map[string]db.Value)
}

func (app *BgpApp) getAppRootObject() *ocbinds.OpenconfigBgp_Bgp {
	deviceObj := (*app.ygotRoot).(*ocbinds.Device)
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
	log.Info("translateGet:bgp:path =", app.pathInfo.Path)
	return nil
}

func (app *BgpApp) translateAction(dbs [db.MaxDB]*db.DB) error {
	return tlerr.NotSupported("unsupported")
}

func (app *BgpApp) translateSubscribe(req translateSubRequest) (translateSubResponse, error) {
	return emptySubscribeResponse(req.path)
}

func (app *BgpApp) processSubscribe(req processSubRequest) (processSubResponse, error) {
	return processSubResponse{}, tlerr.New("not implemented")
}

func (app *BgpApp) processCreate(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	if err = app.processCommon(d, CREATE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}
	return resp, err
}

func (app *BgpApp) processUpdate(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	if err = app.processCommon(d, UPDATE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}
	return resp, err
}

func (app *BgpApp) processReplace(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	if err = app.processCommon(d, REPLACE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}
	return resp, err
}

func (app *BgpApp) processDelete(d *db.DB) (SetResponse, error) {
	var err error
	var resp SetResponse

	if err = app.processCommon(d, DELETE); err != nil {
		log.Error(err)
		resp = SetResponse{ErrSrc: AppErr}
	}
	return resp, err
}

func (app *BgpApp) processGet(dbs [db.MaxDB]*db.DB, fmtType TranslibFmtType) (GetResponse, error) {
	var err error
	var payload []byte

	configDb := dbs[db.ConfigDB]
	err = app.processCommon(configDb, GET)
	if err != nil {
		return GetResponse{Payload: payload, ErrSrc: AppErr}, err
	}

	return generateGetResponse(app.pathInfo.Path, app.ygotRoot, fmtType)
}

func (app *BgpApp) processAction(dbs [db.MaxDB]*db.DB) (ActionResponse, error) {
	return ActionResponse{}, tlerr.New("not implemented")
}

func (app *BgpApp) translateCRUDCommon(d *db.DB, opcode int) ([]db.WatchKeys, error) {
	var err error
	var keys []db.WatchKeys
	log.Info("translateCRUDCommon:bgp:path =", app.pathInfo.Template)

	app.convertOCBgpGlobalsToInternal(opcode)

	return keys, err
}

func (app *BgpApp) processCommon(d *db.DB, opcode int) error {
	var err error
	bgp := app.getAppRootObject()

	log.Infof("processCommon--Path Received: %s", app.pathInfo.Template)

	if isSubtreeRequest(app.pathInfo.Template, "/openconfig-bgp:bgp/global") {
		vrfName := "default"

		switch opcode {
		case CREATE, REPLACE, UPDATE, DELETE:
			err = app.setBgpGlobalsDataInConfigDb(d, opcode)
		case GET:
			err = app.convertDBBgpGlobalsToInternal(d, db.Key{Comp: []string{vrfName}})
			if err != nil {
				return err
			}
			ygot.BuildEmptyTree(bgp.Global)
			app.convertInternalToOCBgpGlobals(vrfName, bgp.Global)
		}
	} else {
		return tlerr.NotSupported("Path not supported")
	}

	return err
}

func (app *BgpApp) setBgpGlobalsDataInConfigDb(d *db.DB, opcode int) error {
	for key, value := range app.bgpGlobalsMap {
		k := db.Key{Comp: []string{key}}
		existingEntry, getErr := d.GetEntry(app.bgpGlobalsTs, k)

		// Return error if GetEntry fails for reasons other than not found
		if getErr != nil && !isNotFoundError(getErr) {
			return getErr
		}

		var err error
		switch opcode {
		case CREATE:
			if existingEntry.IsPopulated() {
				return tlerr.AlreadyExists("BGP global configuration already exists")
			}
			err = d.CreateEntry(app.bgpGlobalsTs, k, value)
		case REPLACE:
			if existingEntry.IsPopulated() {
				err = d.ModEntry(app.bgpGlobalsTs, k, value)
			} else {
				err = d.CreateEntry(app.bgpGlobalsTs, k, value)
			}
		case UPDATE:
			if !existingEntry.IsPopulated() {
				return tlerr.NotFound("BGP global configuration not found")
			}
			err = d.ModEntry(app.bgpGlobalsTs, k, value)
		case DELETE:
			if !existingEntry.IsPopulated() {
				return tlerr.NotFound("BGP global configuration not found")
			}
			err = d.DeleteEntry(app.bgpGlobalsTs, k)
		default:
			return fmt.Errorf("unsupported opcode %d", opcode)
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func (app *BgpApp) convertOCBgpGlobalsToInternal(opcode int) {
	vrfName := "default"

	if opcode == DELETE {
		// For DELETE, just populate the map with the VRF key
		// No need to read from YANG payload since DELETE has no payload
		app.bgpGlobalsMap = make(map[string]db.Value)
		app.bgpGlobalsMap[vrfName] = db.Value{Field: map[string]string{}}
		return
	}

	bgp := app.getAppRootObject()
	if bgp != nil && bgp.Global != nil {
		app.bgpGlobalsMap[vrfName] = db.Value{Field: map[string]string{}}

		if bgp.Global.Config != nil {
			if bgp.Global.Config.As != nil {
				app.bgpGlobalsMap[vrfName].Field["local_asn"] = fmt.Sprint(*bgp.Global.Config.As)
			}
			if bgp.Global.Config.RouterId != nil {
				app.bgpGlobalsMap[vrfName].Field["router_id"] = *bgp.Global.Config.RouterId
			}
		}
	}
}

func (app *BgpApp) convertDBBgpGlobalsToInternal(dbCl *db.DB, key db.Key) error {
	entry, err := dbCl.GetEntry(app.bgpGlobalsTs, key)
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

func (app *BgpApp) convertInternalToOCBgpGlobals(vrfName string, global *ocbinds.OpenconfigBgp_Bgp_Global) {
	if data, ok := app.bgpGlobalsMap[vrfName]; ok {
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
