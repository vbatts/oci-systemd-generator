package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/vbatts/oci-systemd-generator/config"
	"github.com/vbatts/oci-systemd-generator/extract"
	"github.com/vbatts/oci-systemd-generator/layout"
	"github.com/vbatts/oci-systemd-generator/unit"
	"github.com/vbatts/oci-systemd-generator/util"
)

var (
	flConfig   = flag.String("config", "/etc/oci-generator.conf", "configuration for source directory of OCI image-layouts")
	flGenerate = flag.Bool("generate", false, "output a generic configuration file content")
	flDebug    = flag.Bool("debug", false, "enable debug output")
)

func main() {
	var finalErr error
	defer func() {
		if finalErr != nil {
			log.Println("ERROR:", finalErr)
			os.Exit(1)
		}
	}()

	flag.Parse()

	if *flDebug {
		// this is used by Debugf()
		os.Setenv("DEBUG", "1")
	}

	if *flGenerate {
		if _, err := os.Stdout.WriteString(config.DefaultConfig); err != nil {
			finalErr = err
			return
		}
		return
	}

	var cfg *config.OCIGenConfig
	// load default config
	cfg, err := config.LoadConfigFromOptions(strings.NewReader(config.DefaultConfig))
	if err != nil {
		finalErr = err
		return
	}
	// don't fail if the provided config file path does not exist, just use the DefaultConfig
	if *flConfig != "" {
		if _, err := os.Stat(*flConfig); !os.IsNotExist(err) {
			var fh *os.File
			fh, err = os.Open(*flConfig)
			if err != nil {
				finalErr = err
				return
			}
			cfg, err = config.LoadConfigFromOptions(fh)
			if err != nil {
				fh.Close()
				finalErr = err
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
		finalErr = err
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
		finalErr = err
		return
	}

	// If if hasn't been extracted, then apply it to same namespace in extractdir.
	toBeExtracted, err := extract.DetermineNotExtracted(extractedLayouts, manifests)
	if err != nil {
		finalErr = err
		return
	}
	util.Debugf("%d to be extracted", len(toBeExtracted))
	for _, m := range toBeExtracted {
		layout, err := extract.Extract(cfg.ExtractsDir, m)
		if err != nil {
			finalErr = err
			return
		}
		extractedLayouts = append(extractedLayouts, layout)
	}

	// If it has been extracted, check the config's ExecStart()
	// then produce a unit file to os.Args[1,2,3]

	if flag.NArg() == 0 {
		fmt.Println("INFO: no paths provided, not generating unit files.")
		return
	}
	if flag.NArg() > 3 {
		finalErr = fmt.Errorf("Expected 3 or fewer paths, but got %d. See SYSTEMD.GENERATOR(7)", flag.NArg())
		return
	}

	// We'll collect all three, but we're only going to use the dirNormal (for now?)
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
	util.Debugf("%q %q %q", dirNormal, dirEarly, dirLate)

	// Final loops to render a unit file for each extract layout reference which
	// has all the required elements.
	// Required elements are:
	// 1) name (for .service unit file)
	// 2) root directory
	// 3) an ExecStart=
	for _, el := range extractedLayouts {
		refs, err := el.Refs()
		if err != nil {
			finalErr = err
			return
		}
		for _, ref := range refs {
			config, err := ref.Config()
			if err != nil {
				finalErr = err
				return
			}
			execStart := config.ExecStart()
			if execStart == "" {
				fmt.Printf("[INFO] skipping image %s/%s. Empty ExecStart=\n", el.Name, ref.Name)
				continue
			}
			//fmt.Printf("Name: %q; Ref: %q; ExecStart=: %q\n", el.Name, ref.Name, execStart)
			units := unit.DefaultOptions[:]
			u, err := unit.ExecStart(execStart)
			if err != nil {
				finalErr = err
				return
			}
			units = append(units, u)
			u, err = unit.RootDirectory(ref.RootFS())
			if err != nil {
				finalErr = err
				return
			}
			units = append(units, u)

			r := unit.Serialize(units)
			filename := filepath.Join(dirNormal, ref.ReverseDomainNotation()+".service")
			fh, err := os.Create(filename)
			if err != nil {
				finalErr = err
				return
			}
			if _, err := io.Copy(fh, r); err != nil {
				fh.Close()
				finalErr = err
				return
			}
			fh.Close()
			fmt.Printf("wrote %q\n", filename)
		}
	}
}
