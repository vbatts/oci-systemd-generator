package layout

import (
	"path/filepath"
	"strings"

	"github.com/vbatts/oci-systemd-generator/util"
)

// DigestRef for name to digest mapping and validating the blob at the address is
// actually the expected size.
type DigestRef struct {
	Name   string
	Layout *Layout
}

// HashName provides just the hash name portion of the digest string (e.g. "sha256:ed2dca..." -> "sha256")
func (d DigestRef) HashName() string {
	chunks := strings.SplitN(d.Name, digestSeparator, 2)
	if len(chunks) != 2 {
		return ""
	}
	return chunks[0]
}

// Sum provides the hexadecimal portion of the digest string
func (d DigestRef) Sum() string {
	chunks := strings.SplitN(d.Name, digestSeparator, 2)
	if len(chunks) != 2 {
		return ""
	}
	return chunks[1]
}

// Calculate the checksum of the backing blob for this digest, with the prescribed hash.
func (d DigestRef) Calculate() (string, error) {
	fh, err := d.Layout.GetBlob(d)
	if err != nil {
		return "", err
	}
	return util.SumContent(d.HashName(), fh)
}

// IsValid returns whether the backing blob checksum is the same as the referenced digest.
func (d DigestRef) IsValid() (bool, error) {
	sum, err := d.Calculate()
	if err != nil {
		return false, err
	}

	return d.Name == d.HashName()+digestSeparator+sum, nil
}

// "sha256:ed2dca7ba0aa32384f2f5560513dbb0325c8e213b75eb662055e8bd1db7ac974" -> "./sha256/ed2dca7ba0aa32384f2f5560513dbb0325c8e213b75eb662055e8bd1db7ac974"
func digestToPath(digest DigestRef) string {
	chunks := strings.SplitN(digest.Name, digestSeparator, 2)
	if len(chunks) != 2 {
		return ""
	}
	return filepath.Join(chunks[0], chunks[1])
}

// "./sha256/ed2dca7ba0aa32384f2f5560513dbb0325c8e213b75eb662055e8bd1db7ac974" -> "sha256:ed2dca7ba0aa32384f2f5560513dbb0325c8e213b75eb662055e8bd1db7ac974"
func pathToDigest(path string) *DigestRef {
	chunks := strings.Split(filepath.Clean(path), "/")
	if len(chunks) > 1 && chunks[0] == nameBlobs {
		chunks = chunks[1:]
	}
	if len(chunks) != 2 {
		return nil
	}
	return &DigestRef{Name: chunks[0] + digestSeparator + chunks[1]}
}
