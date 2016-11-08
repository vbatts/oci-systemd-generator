package layout

import (
	"opencontainers/image-spec/specs-go/v1"
	"testing"
)

func TestWalk(t *testing.T) {
	layouts, err := WalkForLayouts("../testdata/layouts")
	if err != nil {
		t.Fatal(err)
	}
	expectedLen := 1
	if len(layouts) != expectedLen {
		t.Fatalf("expected %d; got %d", expectedLen, len(layouts))
	}
	expectedName := "tianon/true"
	layout, ok := layouts[expectedName]
	if !ok {
		t.Fatalf("expected to find %q, but did not", expectedName)
	}

	vers, err := layout.OCIVersion()
	if err != nil {
		t.Fatal(err)
	}

	expectedVers := "1.0.0"
	if vers != expectedVers {
		t.Fatalf("expected vers %q; got %q", expectedVers, vers)
	}

	refs, err := layout.Refs()
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Errorf("expected %d; got %d", 1, len(refs))
	}

	for _, ref := range refs {
		desc, err := layout.GetRef(ref)
		if err != nil {
			t.Error(err)
			continue
		}
		manifest, err := ManifestFromDescriptor(layout, desc)
		if err != nil {
			t.Error(err)
			continue
		}

		config, err := manifest.Config()
		if err != nil {
			t.Error(err)
			continue
		}

		chainid, err := config.ChainID()
		if err != nil {
			t.Error(err)
			continue
		}

		expectedChainID := "sha256:3342106d17cf8fc913c462a27e792c09780fac1a34075098f8180398294c976a"
		if chainid.Name != expectedChainID {
			t.Errorf("expected %q; got %q", expectedChainID, chainid.Name)
		}

		for _, desc := range manifest.Manifest.Layers {
			if desc.MediaType != v1.MediaTypeImageLayer {
				continue
			}
			r, err := layout.GetBlob(DigestRef{Name: desc.Digest})
			if err != nil {
				t.Error(err)
				continue
			}
			r.Close()
		}
	}
}
