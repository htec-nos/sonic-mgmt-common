package transformer

import (
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/openconfig/ygot/ygot"
)

func init() {
	XlateFuncBind("DbToYang_hello_world_greeting_xfmr", DbToYang_hello_world_greeting_xfmr)
}

// GET operation - return hardcoded mock data
var DbToYang_hello_world_greeting_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	deviceRoot := (*inParams.ygRoot).(*ocbinds.Device)

	// Access greeting directly - it's a top-level container in your YANG
	if deviceRoot.Greeting == nil {
		ygot.BuildEmptyTree(deviceRoot)
	}

	message := "Hello World!"
	timestamp := "2025-10-23"

	deviceRoot.Greeting.Message = &message
	deviceRoot.Greeting.Timestamp = &timestamp

	return nil
}
