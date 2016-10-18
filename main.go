package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
		if _, err = os.Stdout.WriteString(DefaultConfig); err != nil {
			isErr = true
			return
		}
		return
	}

	var cfg *OCIGenConfig
	// load default config
	cfg, err = LoadConfigFromOptions(strings.NewReader(DefaultConfig))
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
			cfg, err = LoadConfigFromOptions(fh)
			if err != nil {
				fh.Close()
				isErr = true
				return
			}
			fh.Close()
		}
	}
	Debugf("cfg: %q", cfg)

	// Walk cfg.ImageLayoutDir to find directories that have a refs and blobs dir
	var layouts Layouts
	layouts, err = WalkForLayouts(cfg.ImageLayoutDir)
	if err != nil {
		isErr = true
		return
	}

	for name, layout := range layouts {
		// Check the OCI layout version
		if _, err := os.Stat(filepath.Join(cfg.ImageLayoutDir, name, "oci-layout")); os.IsNotExist(err) {
			fmt.Printf("WARN: %q does not have an oci-layout file\n", name)
		}
		refs, err := layout.Refs()
		if err == nil {
			Debugf(name)
			Debugf("\t%q", refs)
		}
	}

	// For each imagelayout determine if it has been extracted.
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
