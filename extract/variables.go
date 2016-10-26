package extract

import "path/filepath"

var (
	nameRef        = "ref"
	nameRootfs     = "rootfs"
	nameNames      = "names"
	nameManifest   = "manifest"
	nameDirs       = "dirs"
	nameChainID    = "chainID"
	nameChainIDDir = filepath.Join(nameDirs, nameChainID)
)
