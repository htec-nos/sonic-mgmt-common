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
	return app.translateCRUCommon(d, CREATE)
}

func (app *BgpApp) translateUpdate(d *db.DB) ([]db.WatchKeys, error) {
	return app.translateCRUCommon(d, UPDATE)
}

func (app *BgpApp) translateReplace(d *db.DB) ([]db.WatchKeys, error) {
	return app.translateCRUCommon(d, REPLACE)
}

func (app *BgpApp) translateDelete(d *db.DB) ([]db.WatchKeys, error) {
	return nil, nil
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

func (app *BgpApp) translateCRUCommon(d *db.DB, opcode int) ([]db.WatchKeys, error) {
	var err error
	var keys []db.WatchKeys
	log.Info("translateCRUCommon:bgp:path =", app.pathInfo.Template)

	app.convertOCBgpGlobalsToInternal()

	return keys, err
}

func (app *BgpApp) processCommon(d *db.DB, opcode int) error {
	var err error
	bgp := app.getAppRootObject()

	log.Infof("processCommon--Path Received: %s", app.pathInfo.Template)

	if isSubtreeRequest(app.pathInfo.Template, "/openconfig-bgp:bgp/global") {
		vrfName := "default"

		switch opcode {
		case CREATE:
			err = app.setBgpGlobalsDataInConfigDb(d, true)
		case REPLACE:
			err = app.setBgpGlobalsDataInConfigDb(d, true)
		case UPDATE:
			err = app.setBgpGlobalsDataInConfigDb(d, false)
		case DELETE:
			err = d.DeleteEntry(app.bgpGlobalsTs, db.Key{Comp: []string{vrfName}})
		case GET:
			err = app.convertDBBgpGlobalsToInternal(d, db.Key{Comp: []string{vrfName}})
			if err != nil {
				return err
			}
			ygot.BuildEmptyTree(bgp.Global)
			app.convertInternalToOCBgpGlobals(vrfName, bgp.Global)
		}
	}

	return err
}

func (app *BgpApp) convertOCBgpGlobalsToInternal() {
	bgp := app.getAppRootObject()
	if bgp != nil && bgp.Global != nil {
		vrfName := "default"
		app.bgpGlobalsMap[vrfName] = db.Value{Field: map[string]string{}}

		if bgp.Global.Config != nil {
			if bgp.Global.Config.As != nil {
				app.bgpGlobalsMap[vrfName].Field["asn"] = fmt.Sprint(*bgp.Global.Config.As)
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

		if asn := data.Get("asn"); asn != "" {
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

func (app *BgpApp) setBgpGlobalsDataInConfigDb(d *db.DB, createFlag bool) error {
	var err error
	for key := range app.bgpGlobalsMap {
		existingEntry, err := d.GetEntry(app.bgpGlobalsTs, db.Key{Comp: []string{key}})

		if createFlag && existingEntry.IsPopulated() {
			return tlerr.AlreadyExists("BGP global configuration already exists")
		}

		if createFlag || (!createFlag && err != nil && !existingEntry.IsPopulated()) {
			err = d.CreateEntry(app.bgpGlobalsTs, db.Key{Comp: []string{key}}, app.bgpGlobalsMap[key])
		} else {
			err = d.ModEntry(app.bgpGlobalsTs, db.Key{Comp: []string{key}}, app.bgpGlobalsMap[key])
		}

		if err != nil {
			return err
		}
	}
	return err
}
