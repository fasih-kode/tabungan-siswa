package http

import (
	stdhttp "net/http"
)

const staticRoutePrefix = "/static/"

// NewStaticAssetHandler membangun handler untuk asset statis aplikasi.
func NewStaticAssetHandler(root string) stdhttp.Handler {
	return stdhttp.StripPrefix(
		staticRoutePrefix,
		stdhttp.FileServer(stdhttp.Dir(root)),
	)
}
