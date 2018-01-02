package imgtmpl

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type cachedImage struct {
	content    image.Image
	accessTime time.Time
}

var (
	imageCache          = make(map[string]*cachedImage)
	imageCacheSize      = 256
	imageCacheMutex     sync.Mutex
	imageLocalPath      = "data/cache/image"
	imageRemoteSync     singleflight.Group
	imageCacheSaveLocal = false
)

func SetImageLocalPath(path string) {
	imageLocalPath = path
}

func SetImageCacheSize(size int) {
	imageCacheSize = size
}

func SetImageCacheSaveLocal(flag bool) {
	imageCacheSaveLocal = flag
}

func getCachedImage(url string) (image.Image, error) {
	if img := getImageFromCache(url); img != nil {
		return img, nil
	}
	res, err, _ := imageRemoteSync.Do(url,
		func() (interface{}, error) {
			img, content, err := getImageFromRemote(url)
			if err != nil {
				return nil, err
			}
			if imageCacheSaveLocal {
				if err := saveLocalImage(url, content); err != nil {
					return nil, err
				}
			}
			imageCacheMutex.Lock()
			defer imageCacheMutex.Unlock()
			updateImageCache(url, img)
			return img, nil
		})
	if err == nil && res != nil {
		return res.(image.Image), nil
	}
	return nil, err
}

func saveLocalImage(url string, content []byte) error {
	filename := fmt.Sprintf("%x", md5.Sum([]byte(url)))
	if _, err := os.Stat(imageLocalPath); err != nil {
		if err := os.MkdirAll(imageLocalPath, 0755); err != nil {
			return err
		}
	}
	return ioutil.WriteFile(path.Join(imageLocalPath, filename), content, 0755)
}

func updateImageCache(url string, img image.Image) {
	if len(imageCache) == imageCacheSize {
		var oldestKey string
		var oldestTime time.Time
		for key, value := range imageCache {
			if oldestKey == "" {
				oldestKey = key
				oldestTime = value.accessTime
			} else if oldestTime.After(value.accessTime) {
				oldestKey = key
				oldestTime = value.accessTime
			}
		}
		delete(imageCache, oldestKey)
	}
	imageCache[url] = &cachedImage{
		content:    img,
		accessTime: time.Now(),
	}
}

func getImageFromCache(url string) image.Image {
	imageCacheMutex.Lock()
	defer imageCacheMutex.Unlock()
	if img, found := imageCache[url]; found && img != nil {
		img.accessTime = time.Now()
		return img.content
	}
	if img := getImageFromLocal(url); img != nil {
		updateImageCache(url, img)
		return img
	}
	return nil
}

func getImageFromLocal(url string) (img image.Image) {
	filePath := path.Join(imageLocalPath, fmt.Sprintf("%x", md5.Sum([]byte(url))))
	if f, err := os.Open(filePath); err == nil {
		if img, _, err = image.Decode(f); err != nil {
			img = nil
		}
	}
	return
}

func getImageFromRemote(url string) (image.Image, []byte, error) {
	if enableLog {
		log.Printf("开始下载远程图片: url=%v", url)
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}
	resp, err := client.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("HTTP应答状态非预期: %d", resp.StatusCode)
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	img, format, err := image.Decode(bytes.NewBuffer(content))
	if err != nil {
		return nil, nil, err
	}
	if enableLog {
		log.Printf("下载远程图片成功: url=%v, length=%d, format=%v",
			url, len(content), format)
	}
	return img, content, nil
}
