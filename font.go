package imgtmpl

import (
	"io/ioutil"
	"log"
	"path"
	"sync"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
)

var (
	fontPath       = "data/fonts"
	fontCache      = make(map[string]*truetype.Font)
	fontCacheMutex sync.Mutex
)

func SetFontPath(path string) {
	fontPath = path
}

func getFont(name string) (*truetype.Font, error) {
	fontCacheMutex.Lock()
	defer fontCacheMutex.Unlock()
	if font, found := fontCache[name]; found && font != nil {
		return font, nil
	} else if content, err := ioutil.ReadFile(path.Join(fontPath, name+".ttf")); err != nil {
		return nil, err
	} else if font, err := freetype.ParseFont(content); err != nil {
		return nil, err
	} else {
		fontCache[name] = font
		if enableLog {
			log.Printf("加载字体成功: name=%v", name)
		}
		return font, nil
	}
}
