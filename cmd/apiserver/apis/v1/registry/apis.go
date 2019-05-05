package registry

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/kataras/iris"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"path"
)

func Register(app *iris.Application) error {
	logrus.Infof("    Registering Agents Mgmt APIs...")
	app.Get("/apis/v1/registry/1.12.5/*", downloadFile)
	return nil
}

func downloadFile(ctx iris.Context) {
	token := ctx.Request().URL.Query().Get("token")
	if token != entities.HTTPDockerImageDownloadToken {
		ctx.StatusCode(http.StatusUnauthorized)
		return
	}
	http.StripPrefix("/apis/v1/", http.FileServer(http.Dir(path.Dir(os.Args[0])))).ServeHTTP(ctx.ResponseWriter(), ctx.Request())
}
