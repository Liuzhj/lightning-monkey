package main

import (
	"encoding/json"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"io/ioutil"
	"net"
	"sync/atomic"
)

type HelloResponse struct {
}

func (a *LightningMonkeyAgent) InitializeWebServer() {
	app := iris.New()
	app.Get("/hello", HealthCheck)
	app.Post("/system/routes", a.GenerateSystemRoutingRules)
	app.Post("/registration/change", a.RegistrationDataChange)
	logrus.Infof("Starting Web Server...")
	app.Run(iris.Addr(fmt.Sprintf("0.0.0.0:%d", *a.arg.ListenPort)))
}

func (a *LightningMonkeyAgent) GenerateSystemRoutingRules(ctx context.Context) {
	req := entities.GenerateSystemRoutingRulesRequest{}
	httpData, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.DeserializeError, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Next()
		return
	}
	err = json.Unmarshal(httpData, &req)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.DeserializeError, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Next()
		return
	}
	logrus.Infof("Setting system routing rules: %#v", req)
	link, err := netlink.LinkByName(*a.arg.UsedEthernetInterface)
	if err != nil {
		logrus.Errorf("Failed to get link information for device %s, error: %s", *a.arg.UsedEthernetInterface, err.Error())
	} else {
		if req.Nodes != nil && len(req.Nodes) > 0 {
			addedRulesCount := 0
			var rr netlink.Route
			//TODO(g0194776): considering add more check logic for routing rules replacement.
			for i := 0; i < len(req.Nodes); i++ {
				_, cidr, err := net.ParseCIDR(req.Nodes[i].PodCIDR)
				if err != nil {
					logrus.Errorf("Failed to parse pod CIDR: %s, error: %s", req.Nodes[i].PodCIDR, err.Error())
					continue
				}
				rr = netlink.Route{
					Gw:        net.ParseIP(req.Nodes[i].NodeIP),
					Dst:       cidr,
					LinkIndex: link.Attrs().Index,
				}
				err = netlink.RouteAdd(&rr)
				if err != nil {
					logrus.Errorf("Failed to add system routing rule, error: %s", err.Error())
				} else {
					addedRulesCount++
				}
			}
			logrus.Warnf("Successfully added system routing rules count: %d", addedRulesCount)
		}
	}
	rsp := entities.Response{ErrorId: 0}
	_, _ = ctx.JSON(rsp)
	ctx.Next()
	return
}

func (a *LightningMonkeyAgent) RegistrationDataChange(ctx context.Context) {
	req := entities.ChangeClusterAndRolesRequest{}
	httpData, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.DeserializeError, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Next()
		return
	}
	err = json.Unmarshal(httpData, &req)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.DeserializeError, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Next()
		return
	}
	logrus.Infof("Changing agent's cluster-id and roles...(OLD: %s, NEW: %s)", req.OldClusterId, req.NewClusterId)
	//change cache.
	a.arg.ClusterId = &req.NewClusterId
	a.arg.IsETCDRole = &req.IsETCDRole
	a.arg.IsMasterRole = &req.IsMasterRole
	a.arg.IsMinionRole = &req.IsMinionRole
	a.arg.IsHARole = &req.IsHARole
	if a.rr != nil {
		a.rr.ClusterID = req.NewClusterId
		a.rr.IsETCDRole = req.IsETCDRole
		a.rr.IsMasterRole = req.IsMasterRole
		a.rr.IsMinionRole = req.IsMinionRole
		a.rr.IsHARole = req.IsHARole
		err = a.saveRecoveryFile()
		if err != nil {
			logrus.Errorf("Failed to save recovery file by updating agent's cluster-id and roles, error: %s", err.Error())
		}
	}
	logrus.Warn("Re-opening agent's registration flag...")
	//reopen registration flag.
	atomic.SwapInt32(&a.hasRegistered, 0)
	rsp := entities.Response{ErrorId: 0}
	_, _ = ctx.JSON(rsp)
	ctx.Next()
	return
}

func HealthCheck(ctx iris.Context) {
	_, _ = ctx.JSON(HelloResponse{})
	ctx.Next()
	return
}
