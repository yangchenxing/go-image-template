package imgtmpl

import (
	"archive/zip"
	"io/ioutil"
)

type Resources map[string][]byte

func (res Resources) loadStringMap(m map[string]string) {
	if m != nil {
		for key, value := range m {
			res[key] = []byte(value)
		}
	}
}

func (res Resources) loadZipFile(path string) error {
	if path == "" {
		return nil
	}
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()
	return res.loadZipReader(r)
}

func (res Resources) loadZipReader(r *zip.ReadCloser) error {
	for _, f := range r.File {
		if rc, err := f.Open(); err != nil {
			return err
		} else if content, err := ioutil.ReadAll(rc); err != nil {
			return err
		} else {
			res[f.Name] = content
		}
	}
	return nil
}

func (res Resources) Get(key string) ([]byte, bool) {
	content, found := res[key]
	return content, found
}
