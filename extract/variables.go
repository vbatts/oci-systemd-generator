package extract

import "path/filepath"

var (
	nameRef        = "config"
	nameRootfs     = "rootfs"
	nameNames      = "names"
	nameConfigs    = "configs"
	nameDirs       = "dirs"
	nameChainID    = "chainID"
	nameChainIDDir = filepath.Join(nameDirs, nameChainID)
)
