package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/go-systemd/unit"
)

var (
	flConfig   = flag.String("config", "/etc/oci-generator.conf", "configuration for source directory of OCI image-layouts")
	flGenerate = flag.Bool("generate", false, "output a generic configuration file content")
	flDebug    = flag.Bool("debug", false, "enable debug output")
)

var DefaultConfig = `
[system]
imagelayoutdir = /home/vbatts/oci/layouts
extractdir = /home/vbatts/oci/extracts
#imagelayoutdir = /var/lib/oci/imagelayout
#extractdir = /var/lib/oci/extract
`

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
	if os.Getenv("DEBUG") != "" {
		fmt.Printf("DEBUG: cfg: %q\n", cfg)
	}

	// Walk cfg.ImageLayoutDir to find directories that have a refs and blobs dir

	var layouts Layout
	layouts, err = WalkForLayouts(cfg.ImageLayoutDir)
	if err != nil {
		isErr = true
		return
	}

	for layout := range layouts {
		if os.Getenv("DEBUG") != "" {
			fmt.Printf("%q\n", layout)
		}
		// Check the OCI layout version
		if _, err := os.Stat(filepath.Join(cfg.ImageLayoutDir, layout, "oci-layout")); os.IsNotExist(err) {
			fmt.Printf("WARN: %q does not have an oci-layout file\n", layout)
		}
		fmt.Println(layout)
	}
	_ = cfg.ExtractDir

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

// OCIGenConfig is the configurations for generating systemd unit files from OCI image layouts
type OCIGenConfig struct {
	ImageLayoutDir string
	ExtractDir     string
}

// LoadConfigFromOptions reads from an INI style set of options
func LoadConfigFromOptions(r io.Reader) (*OCIGenConfig, error) {
	options, err := unit.Deserialize(r)
	if err != nil {
		return nil, err
	}
	cfg := OCIGenConfig{}
	for _, opt := range options {
		if opt.Section == "system" {
			switch opt.Name {
			case "imagelayoutdir":
				cfg.ImageLayoutDir = opt.Value
			case "extractdir":
				cfg.ExtractDir = opt.Value
			}
		}
	}

	return &cfg, nil
}

type Layout map[string]interface{}

func WalkForLayouts(rootpath string) (layout Layout, err error) {
	layout = Layout{}
	err = filepath.Walk(rootpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		var (
			altDir   string
			basename = filepath.Base(path)
			dirname  = filepath.Dir(path)
		)
		switch basename {
		case "refs":
			altDir = filepath.Join(dirname, "blobs")
		case "blobs":
			altDir = filepath.Join(dirname, "refs")
		default:
			return nil
		}

		if altInfo, err := os.Lstat(altDir); err != nil || !altInfo.IsDir() {
			// either this is an error OR it is nil because the directory is not a directory
			return err
		}

		l, err := filepath.Rel(rootpath, dirname)
		if err != nil {
			return err
		}
		if _, ok := layout[l]; !ok {
			layout[l] = nil
		}
		return nil
	})
	return layout, err
}
