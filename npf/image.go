package npf

import (
	"fmt"
	"strconv"
	"strings"
)

// createSrcset renders media objects into srcset data.
func createSrcset(media []MediaObject, urlHandler func(string) string) []string {
	out := make([]string, 0, len(media))
	for _, m := range media {
		out = append(out, fmt.Sprintf("%s %dw", urlHandler(m.URL), m.Width))
	}
	return out
}

// formatImageContainer renders the inner <div class="image-container"> of an
// image block, mirroring npf_renderer.format.image.format_image.
func formatImageContainer(block *ImageBlock, rowLength int, overrideAspect *float64, urlHandler func(string) string, loc Localizer) *Node {
	containerStyle := ""
	imageStyle := ""
	var processed []MediaObject
	var original *MediaObject
	withAspect := true

	for i := range block.Media {
		m := block.Media[i]
		if m.Cropped {
			continue
		}
		processed = append(processed, m)
		if m.HasOriginalDimensions {
			orig := processed[len(processed)-1]
			original = &orig
		}
	}

	if original == nil {
		switch {
		case len(processed) > 0:
			orig := processed[0]
			original = &orig
		case len(block.Media) > 0:
			orig := block.Media[0]
			original = &orig
			imageStyle = fmt.Sprintf("width: %dpx; height: %dpx;", original.Width, original.Height)
			withAspect = false
		default:
			return El("div", "class", "image-container")
		}
	}

	if withAspect {
		if overrideAspect != nil {
			containerStyle = "aspect-ratio: " + pyFloatStr(*overrideAspect) + ";"
		} else {
			ratio := round(float64(original.Width)/float64(original.Height), 4)
			containerStyle = "aspect-ratio: " + pyFloatStr(ratio) + ";"
		}
	}

	container := El("div", "class", "image-container")
	if containerStyle != "" {
		container.SetAttr("style", containerStyle)
	}

	alt := block.AltText
	if alt == "" {
		alt = loc.Translate("generic_image_alt_text", nil)
	}

	img := El("img")
	img.SetAttr("src", urlHandler(original.URL))
	img.SetAttr("srcset", strings.Join(createSrcset(processed, urlHandler), ", "))
	img.SetAttr("class", "image")
	img.SetAttr("loading", "lazy")
	img.SetAttr("alt", alt)
	img.SetAttr("sizes", fmt.Sprintf("(max-width: 540px) %dvh, %dpx", 100/rowLength, 540/rowLength))
	if imageStyle != "" {
		img.SetAttr("style", imageStyle)
	}
	container.Add(img)
	return container
}

// round rounds f to n decimal places the way Python's round() does
// (half-to-even), then returns it as a float.
func round(f float64, n int) float64 {
	s := strconv.FormatFloat(f, 'f', n, 64)
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
