package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"vb/oci-systemd-generator/extract"

	"github.com/vbatts/oci-systemd-generator/config"
	"github.com/vbatts/oci-systemd-generator/layout"
	"github.com/vbatts/oci-systemd-generator/util"
)

var (
	flConfig   = flag.String("config", "/etc/oci-generator.conf", "configuration for source directory of OCI image-layouts")
	flGenerate = flag.Bool("generate", false, "output a generic configuration file content")
	flDebug    = flag.Bool("debug", false, "enable debug output")
)

func main() {
	var isErr bool
	var err error
	defer func() {
		if isErr {
			log.Println("ERROR: ", err)
			os.Exit(1)
		}
	}()

	flag.Parse()

	if *flDebug {
		// this is used by Debugf()
		os.Setenv("DEBUG", "1")
	}

	if *flGenerate {
		if _, err = os.Stdout.WriteString(config.DefaultConfig); err != nil {
			isErr = true
			return
		}
		return
	}

	var cfg *config.OCIGenConfig
	// load default config
	cfg, err = config.LoadConfigFromOptions(strings.NewReader(config.DefaultConfig))
	if err != nil {
		isErr = true
		return
	}
	// don't fail if the provided config file path does not exist, just use the DefaultConfig
	if *flConfig != "" {
		if _, err := os.Stat(*flConfig); !os.IsNotExist(err) {
			var fh *os.File
			fh, err = os.Open(*flConfig)
			if err != nil {
				isErr = true
				return
			}
			cfg, err = config.LoadConfigFromOptions(fh)
			if err != nil {
				fh.Close()
				isErr = true
				return
			}
			fh.Close()
		}
	}
	util.Debugf("cfg: %q", cfg)

	// Walk cfg.ImageLayoutDir to find directories that have a refs and blobs dir
	var layouts layout.Layouts
	layouts, err = layout.WalkForLayouts(cfg.ImageLayoutDir)
	if err != nil {
		isErr = true
		return
	}

	// Check all the layouts available
	manifests := []*layout.Manifest{}
layoutLoop:
	for name, l := range layouts {
		// Check the OCI layout version
		if _, err := os.Stat(filepath.Join(cfg.ImageLayoutDir, name, "oci-layout")); os.IsNotExist(err) {
			fmt.Printf("WARN: %q does not have an oci-layout file\n", name)
		}
		refs, err := l.Refs()
		if err != nil {
			continue
		}
		blobs, err := l.Blobs()
		if err != nil {
			continue
		}
		util.Debugf(name)
		util.Debugf("\tnum blobs: %d", len(blobs))
		allValid := true
		for _, blob := range blobs {
			valid, err := blob.IsValid()
			if err != nil {
				break
			}
			if !valid {
				util.Debugf("\tblob failed: %q", blob.Name)
				allValid = false
			}
		}
		if allValid {
			util.Debugf("\tblob checksums: PASS")
		} else {
			util.Debugf("\tblob checksums: FAILED")
			continue layoutLoop
		}

		util.Debugf("\trefs:")
		for _, ref := range refs {
			desc, err := l.GetRef(ref)
			if err != nil {
				continue
			}
			util.Debugf("\t\t%s: %#v", ref, desc)
			manifest, err := layout.ManifestFromDescriptor(l, desc)
			if err != nil {
				if err == layout.ErrUnsupportedMediaType {
					log.Println(err)
					continue
				}
				log.Printf("%q: %s", ref, err)
				break
			}
			manifest.Ref = ref
			manifests = append(manifests, manifest)
		}
	}

	extractedLayouts, err := extract.WalkForExtracts(cfg.ExtractsDir)
	if err != nil && err != extract.ErrNoExtracts {
		isErr = true
		return
	}

	// If if hasn't been extracted, then apply it to same namespace in extractdir.
	toBeExtracted, err := extract.DetermineNotExtracted(extractedLayouts, manifests)
	if err != nil {
		isErr = true
		return
	}
	util.Debugf("%d to be extracted", len(toBeExtracted))
	// XXX

	// If it has been extracted, check the config's ExecStart()
	// then produce a unit file to os.Args[1,2,3]

	if flag.NArg() == 0 {
		fmt.Println("INFO: no paths provided, not generating unit files.")
		return
	}
	if flag.NArg() > 3 {
		isErr = true
		err = fmt.Errorf("Expected 3 or fewer paths, but got %d. See SYSTEMD.GENERATOR(7)", flag.NArg())
		return
	}

	var dirNormal, dirEarly, dirLate string
	if flag.NArg() == 3 {
		dirLate = flag.Args()[2]
	}
	if flag.NArg() >= 2 {
		dirEarly = flag.Args()[1]
	}
	if flag.NArg() >= 1 {
		dirNormal = flag.Args()[0]
	}

	fmt.Println(dirNormal, dirEarly, dirLate)
}
