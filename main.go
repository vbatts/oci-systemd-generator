package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/image-spec/specs-go/v1"
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
	manifests := []layout.Manifest{}
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
			if desc.MediaType != v1.MediaTypeImageManifest && desc.MediaType != v1.MediaTypeImageManifestList {
				log.Printf("%q: unsupported Medatype %q, skipping", ref, desc.MediaType)
				break
			}
			if desc.MediaType == v1.MediaTypeImageManifestList {
				log.Println("TODO: add support for manifest list")
				continue
			}
			manifestFH, err := l.GetBlob(layout.DigestRef{Name: desc.Digest})
			if err != nil {
				log.Println(err)
				break
			}
			manifest := v1.Manifest{}
			dec := json.NewDecoder(manifestFH)
			if err := dec.Decode(&manifest); err != nil {
				log.Println(err)
				manifestFH.Close()
				break
			}
			manifestFH.Close()
			util.Debugf("%#v", manifest)
			manifests = append(manifests, layout.Manifest{Layout: l, Ref: ref, Manifest: &manifest})
		}
	}

	// For each imagelayout determine if it has been extracted.
	//for _, manifest := range manifests {
	//}

	// If if hasn't beenen extracted, then apply it to same namespace in extractdir.
	// If it has been extracted, then produce a unit file to os.Args[1,2,3]

	if flag.NArg() == 0 {
		fmt.Println("INFO: no paths provided, not generating unit files.")
		return
	}
	if flag.NArg() > 3 {
		isErr = true
		err = fmt.Errorf("Expected 3 or fewer paths, but got %d. See SYSTEMD.GENERATOR(7)", flag.NArg())
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
