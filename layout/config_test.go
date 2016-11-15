package layout

import (
	"testing"

	"github.com/opencontainers/image-spec/specs-go/v1"
)

func TestChainID(t *testing.T) {
	expect := "sha256:26ab43e7d8f84043e608d1be92d9918a8aa6860489b6f221f29bc49d0081c3c3"
	c := Config{
		ImageConfig: &v1.Image{
			RootFS: v1.RootFS{
				Type: "layers",
				DiffIDs: []string{
					"sha256:8b2564fc821995bf961877c06e676591297178a4438293ec1c2e20f66855d3a8",
					"sha256:ecf239b31bd1723014debab23dd82b3b4b87114e3254370c366f7f712fb9b075",
					"sha256:eab89ad04ba05c973e3fb95f9b10a2875a18779d9977743c779c8238aa3fd742",
				},
			},
		},
	}

	/*
		in bash the equivalent is:
		```bash
		sha256() {
			echo "sha256:$(openssl sha256 ${1+"$@"} | awk '{ print $2 }')";
		}
		step=$(echo -n sha256:8b2564fc821995bf961877c06e676591297178a4438293ec1c2e20f66855d3a8 sha256:ecf239b31bd1723014debab23dd82b3b4b87114e3254370c366f7f712fb9b075 | sha256)
		step=$(echo -n ${step} sha256:eab89ad04ba05c973e3fb95f9b10a2875a18779d9977743c779c8238aa3fd742 | sha256)
		echo ${step}
		# sha256:26ab43e7d8f84043e608d1be92d9918a8aa6860489b6f221f29bc49d0081c3c3
		```
	*/

	chainDigest, err := c.ChainID()
	if err != nil {
		t.Fatal(err)
	}
	if chainDigest.Name != expect {
		t.Fatalf("expected %q; got %q", expect, chainDigest.Name)
	}
}
