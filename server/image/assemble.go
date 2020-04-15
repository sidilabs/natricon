package image

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"strings"

	svg "github.com/ajstarks/svgo"
	"github.com/appditto/natricon/color"
	"github.com/golang/glog"
)

const defaultSize = 1000  // Default SVG width/height attribute
const opacityLower = 0.15 // Minimum lower opacity threshold
const opacityUpper = 0.6  // Maximum upper opacity threshold

type SVG struct {
	Width  int    `xml:"width,attr"`
	Height int    `xml:"height,attr"`
	Doc    string `xml:",innerxml"`
}

func CombineSVG(accessories Accessories) ([]byte, error) {
	var (
		body     SVG
		hair     SVG
		mouth    SVG
		eye      SVG
		backHair SVG
	)
	// Parse all SVG assets
	if err := xml.Unmarshal(accessories.BodyAsset.SVGContents, &body); err != nil {
		glog.Fatalf("Unable to parse body SVG %v", err)
		return nil, err
	}
	if err := xml.Unmarshal(accessories.HairAsset.SVGContents, &hair); err != nil {
		glog.Fatalf("Unable to parse hair SVG %v", err)
		return nil, err
	}
	if err := xml.Unmarshal(accessories.MouthAsset.SVGContents, &mouth); err != nil {
		glog.Fatalf("Unable to parse mouth SVG %v", err)
		return nil, err
	}
	if err := xml.Unmarshal(accessories.EyeAsset.SVGContents, &eye); err != nil {
		glog.Fatalf("Unable to parse eye SVG %v", err)
		return nil, err
	}
	if accessories.BackHairAsset != nil {
		if err := xml.Unmarshal(accessories.BackHairAsset.SVGContents, &backHair); err != nil {
			glog.Fatalf("Unable to parse back hair SVG %v", err)
			return nil, err
		}
	}
	// Create new SVG writer
	var b bytes.Buffer
	canvas := svg.New(&b)
	canvas.Startraw(fmt.Sprintf("viewbox=\"0 0 %d %d\"", defaultSize, defaultSize))
	// Add back hair first
	if accessories.BackHairAsset != nil {
		canvas.Gid("backhair")
		if accessories.HairAsset.HairColored {
			backHair.Doc = strings.ReplaceAll(backHair.Doc, "#FF0000", accessories.HairColor.ToHTML(true))
			backHair.Doc = strings.ReplaceAll(backHair.Doc, "fill-opacity=\"0.15\"", fmt.Sprintf("fill-opacity=\"%f\"", GetTargetOpacity(accessories.HairColor.ToHSV())))
		}
		io.WriteString(canvas.Writer, backHair.Doc)
		canvas.Gend()
	}
	// Body group
	canvas.Gid("body")
	if accessories.BodyAsset.BodyColored {
		body.Doc = strings.ReplaceAll(body.Doc, "#00FFFF", accessories.BodyColor.ToHTML(true))
		body.Doc = strings.ReplaceAll(body.Doc, "fill-opacity=\"0.15\"", fmt.Sprintf("fill-opacity=\"%f\"", GetTargetOpacity(accessories.BodyColor.ToHSV())))
	}
	io.WriteString(canvas.Writer, body.Doc)
	canvas.Gend()
	// Hair Group
	canvas.Gid("hair")
	if accessories.HairAsset.HairColored {
		hair.Doc = strings.ReplaceAll(hair.Doc, "#FF0000", accessories.HairColor.ToHTML(true))
		hair.Doc = strings.ReplaceAll(hair.Doc, "fill-opacity=\"0.15\"", fmt.Sprintf("fill-opacity=\"%f\"", GetTargetOpacity(accessories.HairColor.ToHSV())))
	}
	io.WriteString(canvas.Writer, hair.Doc)
	canvas.Gend()
	// Mouth group
	canvas.Gid("mouth")
	if accessories.HairAsset.HairColored {
		mouth.Doc = strings.ReplaceAll(mouth.Doc, "#FFFF00", accessories.HairColor.ToHTML(true))
	}
	io.WriteString(canvas.Writer, mouth.Doc)
	canvas.Gend()
	// Eye group
	canvas.Gid("eye")
	io.WriteString(canvas.Writer, eye.Doc)
	canvas.Gend()
	// End document
	canvas.End()

	return b.Bytes(), nil
}

func GetTargetOpacity(color color.HSV) float64 {
	brightness := color.V
	ret := (brightness-MinBrightness)*(opacityUpper-opacityLower)/(1.0-MinBrightness) + opacityLower
	// Return result rounded to 2 places
	return math.Round(ret*100) / 100
}
