package imgtmpl

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"path"

	"github.com/mitchellh/mapstructure"
	"golang.org/x/image/draw"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/riff"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type ImageTemplate struct {
	Width           int                      `json:"width"`
	Height          int                      `json:"height"`
	BackgroundColor string                   `json:"background_color"`
	Components      []map[string]interface{} `json:"components"`
	Resources       map[string]string        `json:"resources"`
	ResourceFile    string                   `json:"resource_file"`
	background      image.Image
	components      []Component
	path            string
}

func (tmpl *ImageTemplate) Render(params map[string]string) (draw.Image, error) {
	dst := image.NewRGBA(image.Rect(0, 0, tmpl.Width, tmpl.Height))
	if tmpl.background != nil {
		draw.Copy(dst, image.ZP, tmpl.background, dst.Bounds(), draw.Src, nil)
	}
	for i, component := range tmpl.components {
		if err := component.Render(dst, params); err != nil {
			return dst, fmt.Errorf("渲染第%d个组件失败: %s", i, err.Error())
		}
	}
	return dst, nil
}

type Component interface {
	Init(Resources) error
	Render(draw.Image, map[string]string) error
}

var (
	componentFactories = make(map[string]func() Component)
)

func LoadImageTemplateZipFile(path string) (*ImageTemplate, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	resources := make(Resources)
	if err := resources.loadZipReader(r); err != nil {
		return nil, err
	}
	tmpl := &ImageTemplate{
		path: path,
	}
	content, found := resources.Get("template.json")
	if !found {
		return nil, fmt.Errorf("未找到template.json")
	}
	if err := tmpl.loadContent(content, resources); err != nil {
		return nil, err
	}
	return tmpl, nil
}

func LoadImageTemplateJsonFile(path string) (*ImageTemplate, error) {
	tmpl := &ImageTemplate{
		path: path,
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if err := tmpl.loadContent(content, nil); err != nil {
		return nil, err
	}
	return tmpl, nil
}

func (tmpl *ImageTemplate) loadContent(content []byte, resources Resources) error {
	if err := json.Unmarshal(content, tmpl); err != nil {
		return err
	}
	if len(tmpl.Components) == 0 {
		return fmt.Errorf("缺少components字段")
	}
	if tmpl.BackgroundColor != "" {
		backgroundColor, err := parseColor(tmpl.BackgroundColor)
		if err != nil {
			return fmt.Errorf("背景色格式错误: %s", err.Error())
		}
		tmpl.background = image.NewUniform(backgroundColor)
	}
	// 初始化资源
	if resources == nil && tmpl.ResourceFile != "" {
		resources = make(Resources)
		resourceFilePath := path.Join(path.Dir(tmpl.path), tmpl.ResourceFile)
		if err := resources.loadZipFile(resourceFilePath); err != nil {
			return err
		}
	}
	if resources == nil {
		resources = make(Resources)
	}
	resources.loadStringMap(tmpl.Resources)
	// 初始化组件
	tmpl.components = make([]Component, len(tmpl.Components))
	for i, comp := range tmpl.Components {
		if typ, found := comp["type"]; !found {
			return fmt.Errorf("第%d个组件缺少Type字段", i)
		} else if typ, ok := typ.(string); !ok {
			return fmt.Errorf("第%d个组件Type字段类型错误", i)
		} else if factory, found := componentFactories[typ]; !found || factory == nil {
			return fmt.Errorf("第%d个组件类型未知", i)
		} else {
			component := factory()
			if err := mapstructure.Decode(comp, component); err != nil {
				return fmt.Errorf("第%d个组件解码错误: %s", i, err.Error())
			}
			if err := component.Init(resources); err != nil {
				return fmt.Errorf("第%d个组件初始化出错: %s", i, err.Error())
			}
			tmpl.components[i] = component
		}
	}
	// 初始化完成后清空原始配置，等待内存回收
	tmpl.Components = nil
	tmpl.Resources = nil
	return nil
}
