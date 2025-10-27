package transformer

import (
	log "github.com/golang/glog"
)

func init() {
	XlateFuncBind("bgp_global_key_xfmr", DbToYang_bgp_global_key_xfmr)
	XlateFuncBind("YangToDb_bgp_global_key_xfmr", YangToDb_bgp_global_key_xfmr)
	XlateFuncBind("DbToYang_bgp_global_key_xfmr", DbToYang_bgp_global_key_xfmr)
}

const (
	BGP_GLOBALS_TABLE = "BGP_GLOBALS"
)

// YangToDb key transformer - converts YANG request to DB key
var YangToDb_bgp_global_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	var err error
	var vrfName string = "default"

	log.Infof("YangToDb_bgp_global_key_xfmr: uri=%v", inParams.uri)

	// For now, we only support default VRF
	return vrfName, err
}

// DbToYang key transformer - converts DB key to YANG format
var DbToYang_bgp_global_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	var err error
	result := make(map[string]interface{})

	log.Infof("DbToYang_bgp_global_key_xfmr: key=%v", inParams.key)

	// OpenConfig BGP global is a container (not a list), so no keys to return
	return result, err
}
