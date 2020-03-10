package imports

import (
	"github.com/forensicanalysis/forensicstore/gojsonlite"
	"github.com/imdario/mergo"
	"io"
	"path/filepath"
	"strings"
)

// JsonLite merges another JSONLite into this one.
func JsonLite(db *gojsonlite.JSONLite, url string) (err error) {
	// TODO: import items with "_path" on sublevel"…
	// TODO: import does not need to unflatten and flatten

	importStore, err := gojsonlite.New(url, "")
	if err != nil {
		return err
	}
	items, err := importStore.All()
	if err != nil {
		return err
	}
	for _, item := range items {
		for field := range item {
			item := item
			if strings.HasSuffix(field, "_path") {
				dstPath, writer, err := db.StoreFile(item[field].(string))
				if err != nil {
					return err
				}
				reader, err := importStore.Open(filepath.Join(url, item[field].(string)))
				if err != nil {
					return err
				}
				if _, err = io.Copy(writer, reader); err != nil {
					return err
				}
				if err := mergo.Merge(&item, gojsonlite.Item{field: dstPath}); err != nil {
					return err
				}
			}
		}
		_, err = db.Insert(item)
		if err != nil {
			return err
		}
	}
	return err
}
