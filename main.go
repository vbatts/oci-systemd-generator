package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/coreos/go-systemd/unit"
)

var (
	flConfig   = flag.String("config", "/etc/oci-generator.conf", "configuration for source directory of OCI image-layouts")
	flGenerate = flag.Bool("generate", false, "output a generic configuration file content")
	flDebug    = flag.Bool("debug", false, "enable debug output")
)

var DefaultConfig = []*unit.UnitOption{
	&unit.UnitOption{
		Section: "system",
		Name:    "imagelayoutdir",
		Value:   "/home/vbatts/oci/layouts",
		//Value:   "/var/lib/oci/imagelayout",
	},
	&unit.UnitOption{
		Section: "system",
		Name:    "extractdir",
		Value:   "/home/vbatts/oci/extracts",
		//Value:   "/var/lib/oci/extract",
	},
}

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
		rdr := unit.Serialize(DefaultConfig)
		if _, err = io.Copy(os.Stdout, rdr); err != nil {
			isErr = true
			return
		}
		return
	}

	var options []*unit.UnitOption = DefaultConfig
	// don't fail if the provided config file path does not exist, just use the DefaultConfig
	if *flConfig != "" {
		if _, err := os.Stat(*flConfig); !os.IsNotExist(err) {
			var fh *os.File
			fh, err = os.Open(*flConfig)
			if err != nil {
				isErr = true
				return
			}
			options, err = unit.Deserialize(fh)
			if err != nil {
				fh.Close()
				println(err.Error())
				isErr = true
				return
			}
			fh.Close()
		}
	}
	if os.Getenv("DEBUG") != "" {
		fmt.Printf("DEBUG: options: %q\n", options)
	}

	var imagelayoutdir, extractdir string
	for _, opt := range options {
		if opt.Section == "system" {
			switch opt.Name {
			case "imagelayoutdir":
				imagelayoutdir = opt.Value
			case "extractdir":
				extractdir = opt.Value
			}
		}
	}
	// Walk imagelayoutdir to find directories that have a refs and blobs dir

	var layout Layout
	layout, err = WalkForLayouts(imagelayoutdir)
	if err != nil {
		isErr = true
		return
	}
	if os.Getenv("DEBUG") != "" {
		fmt.Printf("%q\n", layout)
	}

	_ = extractdir

	// For each imagelayout determine if it has been extracted.
	// If if hasn't beenen extracted, then apply it to same namespace in extractdir.
	// If it has been extracted, then produce a unit file to os.Args[1,2,3]

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
