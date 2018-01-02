package imgtmpl

import (
	"bytes"
	"fmt"
	"image"

	"golang.org/x/image/draw"
)

func init() {
	componentFactories["fixed_image"] = func() Component {
		return new(FixedImage)
	}
	componentFactories["clip_image"] = func() Component {
		return new(ClipImage)
	}
}

type FixedImage struct {
	Point  image.Point
	Source string
	source image.Image
}

func (img *FixedImage) Init(res Resources) (err error) {
	if content, found := res.Get(img.Source); found {
		if bytes.HasPrefix(content, []byte("data:image/")) {
			img.source, err = loadBase64Image(content)
		} else {
			img.source, _, err = image.Decode(bytes.NewBuffer(content))
		}
	}
	return
}

func (img *FixedImage) Render(dst draw.Image, _ map[string]string) (err error) {
	if img.source == nil {
		if img.source, err = getCachedImage(img.Source); err != nil {
			return
		}
	}
	draw.Copy(dst, img.Point, img.source, img.source.Bounds(), draw.Over, nil)
	return nil
}

type ClipImage struct {
	Bounds image.Rectangle
	Source string
	Clip   image.Rectangle
}

func (img ClipImage) Init(_ Resources) error {
	return nil
}

func (img ClipImage) Render(dst draw.Image, params map[string]string) error {
	sourceURL, found := params[img.Source]
	if !found {
		return fmt.Errorf("缺少参数: %s", img.Source)
	}
	src, _, err := getImageFromRemote(sourceURL)
	if err != nil {
		return err
	}
	draw.ApproxBiLinear.Scale(dst, img.Bounds, src, img.Clip, draw.Src, nil)
	return nil
}
