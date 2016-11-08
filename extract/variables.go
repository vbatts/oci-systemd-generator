package extract

import "path/filepath"

var (
	nameRef        = "ref"
	nameRootfs     = "rootfs"
	nameNames      = "names"
	nameConfigs    = "configs"
	nameDirs       = "dirs"
	nameChainID    = "chainID"
	nameChainIDDir = filepath.Join(nameDirs, nameChainID)
)
