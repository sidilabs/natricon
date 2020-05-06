package image

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
)

type IllustrationType string
type Sex string

const (
	Body     IllustrationType = "body"
	Hair     IllustrationType = "hair-front"
	HairBack IllustrationType = "hair-back"
	Mouth    IllustrationType = "mouth"
	Eye      IllustrationType = "eyes"
	Male     Sex              = "M"
	Female   Sex              = "F"
	Neutral  Sex              = "N"
)

type Asset struct {
	FileName         string           // File name of asset
	IllustrationPath string           // Full path of illustration on the file system
	Type             IllustrationType // Type of illustration (body, hair, mouth, eye)
	SVGContents      []byte           // Full contents of SVG asset
	HairColored      bool             // Whether this asset should be colored the same as hair color
	BodyColored      bool             // Whether this asset should be colored the same as body color
	Sex              Sex              // The Sex condition of this asset
}

// getIllustrationPath - get full path of image
func getIllustrationPath(illustration string, iType IllustrationType) string {
	wd, err := os.Getwd()
	if err != nil {
		panic("Can't get working directory")
	}

	fPath := path.Join(wd, "assets", "illustrations", string(iType), illustration)
	if _, err := os.Stat(fPath); err != nil {
		// File does not exist
		panic(fmt.Sprintf("File %s does not exist", fPath))
	}
	return fPath
}

// getSex - get Sex based on image name
func getSex(name string) Sex {
	if strings.Contains(name, "_f") {
		return Female
	} else if strings.Contains(name, "_m") {
		return Male
	}
	return Neutral
}

// Singleton to keep assets loaded in memory
type assetManager struct {
	bodyAssets     []Asset
	hairAssets     []Asset
	hairBackAssets []Asset
	mouthAssets    []Asset
	eyeAssets      []Asset
}

var singleton *assetManager
var once sync.Once

func GetAssets() *assetManager {
	once.Do(func() {
		var err error
		// Load body assets
		var bodyAssets []Asset
		for _, ba := range BodyIllustrations {
			var a Asset
			err = json.Unmarshal(ba, &a)
			bodyAssets = append(bodyAssets, a)
		}
		// Load hair assets
		var hairAssets []Asset
		for _, ha := range HairIllustrations {
			var a Asset
			err = json.Unmarshal(ha, &a)
			hairAssets = append(hairAssets, a)
		}
		// Load hair back assets
		var hairBackAssets []Asset
		for _, ha := range HairBackIllustrations {
			var a Asset
			err = json.Unmarshal(ha, &a)
			hairBackAssets = append(hairBackAssets, a)
		}
		// Load mouth assets
		var mouthAssets []Asset
		for _, ma := range MouthIllustrations {
			var a Asset
			err = json.Unmarshal(ma, &a)
			mouthAssets = append(mouthAssets, a)
		}
		// Load eye assets
		var eyeAssets []Asset
		for _, ea := range EyeIllustrations {
			var a Asset
			err = json.Unmarshal(ea, &a)
			eyeAssets = append(eyeAssets, a)
		}
		if err != nil {
			panic("Failed to decode assets")
		}
		// Create object
		singleton = &assetManager{
			bodyAssets:     bodyAssets,
			hairAssets:     hairAssets,
			hairBackAssets: hairBackAssets,
			mouthAssets:    mouthAssets,
			eyeAssets:      eyeAssets,
		}
	})
	return singleton
}

// GetNBodyAssets - get # of body assets
func (sm *assetManager) GetNBodyAssets() int {
	return len(sm.bodyAssets)
}

// GetBodyAssets - get complete list of hair assets
func (sm *assetManager) GetBodyAssets() []Asset {
	return sm.bodyAssets
}

// GetNHairAssets - get # of hair assets
func (sm *assetManager) GetNHairAssets() int {
	return len(sm.hairAssets)
}

// GetHairAssets - get complete list of hair assets
func (sm *assetManager) GetHairAssets(sex Sex) []Asset {
	var ret []Asset
	for _, v := range sm.hairAssets {
		if sex == Neutral {
			ret = append(ret, v)
		} else if v.Sex == sex || v.Sex == Neutral {
			ret = append(ret, v)
		}
	}
	return ret
}

// GetBackHairAssets - get complete list of hair assets
func (sm *assetManager) GetBackHairAssets() []Asset {
	return sm.hairBackAssets
}

// GetMouthAssets - Get mouth assets
func (sm *assetManager) GetMouthAssets(sex Sex) []Asset {
	var ret []Asset
	for _, v := range sm.mouthAssets {
		if sex == Neutral {
			ret = append(ret, v)
		} else if v.Sex == sex || v.Sex == Neutral {
			ret = append(ret, v)
		}
	}
	return ret
}

// GetEyeAssets - Get eye asset list
func (sm *assetManager) GetEyeAssets(sex Sex) []Asset {
	var ret []Asset
	for _, v := range sm.eyeAssets {
		if sex == Neutral {
			ret = append(ret, v)
		} else if v.Sex == sex || v.Sex == Neutral {
			ret = append(ret, v)
		}
	}
	return ret
}
