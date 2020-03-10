package export

// ExportJSONLite clones the JSONLite database.
/*
func ExportJSONLite(db *gojsonlite.JSONLite, url string) (err error) {
	err = db.cursor.Close()
	if err != nil {
		return err
	}
	remoteFS, remoteFolder, _ := toFS(url)
	err = copy.Directory(db.localFS, remoteFS, db.localStoreFolder, remoteFolder)
	if err != nil {
		return err
	}
	db.cursor, err = sql.Open("sqlite3", db.localDBFile)
	return err
}
*/
