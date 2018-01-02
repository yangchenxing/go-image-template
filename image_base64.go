package imgtmpl

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
)

func loadBase64Image(content []byte) (image.Image, error) {
	parts := bytes.SplitN(content, []byte(","), 2)
	b64decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewBuffer(parts[1]))
	format := string(parts[0][11 : len(parts[0])-7])
	switch format {
	case "png":
		return png.Decode(b64decoder)
	case "jpeg":
		return jpeg.Decode(b64decoder)
	case "gif":
		return gif.Decode(b64decoder)
	}
	return nil, fmt.Errorf("未知图像格式: %s", format)
}
