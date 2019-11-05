package debug

import (
	"github.com/kataras/iris"
	"github.com/sirupsen/logrus"
	"runtime"
)

func Register(app *iris.Application) error {
	logrus.Infof("    Registering Troubleshooting  APIs...")
	app.Get("/debug/dump", GetStrackTraceDump)
	return nil
}

func GetStrackTraceDump(ctx iris.Context) {
	data := make([]byte, 1024*1024*20)
	n := runtime.Stack(data, true)
	_, _ = ctx.WriteString(string(data[:n]))
	ctx.Next()
	return
}
