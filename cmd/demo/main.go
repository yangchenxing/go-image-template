package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"image/png"
	"io/ioutil"
	"log"
	"path"

	"github.com/yangchenxing/go-image-template"
)

func main() {
	//imgtmpl.EnableLog()
	tmplFile := flag.String("tmpl", "tmpl.json", "模板文件路径")
	paramFile := flag.String("param", "param.json", "参数文件路径")
	outFile := flag.String("out", "out.png", "输出文件路径")
	flag.Parse()
	var tmpl *imgtmpl.ImageTemplate
	if content, err := ioutil.ReadFile(*tmplFile); err != nil {
		panic(err)
	} else if tmpl, err = imgtmpl.LoadImageTemplate(content, path.Dir(*tmplFile)); err != nil {
		panic(err)
	}
	log.Println("加载模板完成")
	var params map[string]string
	if content, err := ioutil.ReadFile(*paramFile); err != nil {
		panic(err)
	} else if err := json.Unmarshal(content, &params); err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	if dst, err := tmpl.Render(params); err != nil {
		panic(err)
	} else if png.Encode(&buf, dst); err != nil {
		panic(err)
	} else if err := ioutil.WriteFile(*outFile, buf.Bytes(), 0755); err != nil {
		panic(err)
	}
	log.Println("渲染完成")
}
