package plugins

import (
	"encoding/json"
	"github.com/Velocidex/ordereddict"
	"github.com/forensicanalysis/forensicstore/goforensicstore"
	"path"
	"strings"
	"www.velocidex.com/golang/evtx"
)

func init() {
	Plugins["eventlogs"] = &EventlogsPlugin{}
}

type EventlogsPlugin struct{}

func (*EventlogsPlugin) Run(url string, data Data) error {
	store, err := goforensicstore.NewJSONLite(url)
	if err != nil {
		return err
	}

	fileItems, err := store.Select("file")
	if err != nil {
		return err
	}

	for _, item := range fileItems {
		if name, ok := item["name"]; ok {
			if name, ok := name.(string); ok {
				if strings.HasSuffix(name, ".evtx") {
					if exportPath, ok := item["export_path"]; ok {
						if exportPath, ok := exportPath.(string); ok {
							file, err := store.Open(path.Join(url, exportPath))
							if err != nil {
								return err
							}

							chunks, err := evtx.GetChunks(file)
							if err != nil {
								return err
							}

							for _, chunk := range chunks {
								records, err := chunk.Parse(int(chunk.Header.FirstEventRecID))
								if err != nil {
									return err
								}

								for _, i := range records {
									eventMap, ok := i.Event.(*ordereddict.Dict)
									if ok {
										event, ok := ordereddict.GetMap(eventMap, "Event")
										if !ok {
											continue
										}

										event.Set("type", "eventlog")
										// self.maybeExpandMessage(event)

										serialized, err := json.MarshalIndent(event, " ", " ")
										if err != nil {
											return err
										}

										var item map[string]interface{}
										err = json.Unmarshal(serialized, &item)
										if err != nil {
											return err
										}

										_, err = store.Insert(item)
										if err != nil {
											return err
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}
