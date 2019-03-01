package mongodb

import (
	"errors"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/certs"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"log"
	"time"
)

type MongoDBStorageDriver struct {
	connectionStr string
	rootSession   *mgo.Session
	strongSession *mgo.Session
}

func (mc *MongoDBStorageDriver) InitializeByConfig(connectionStr string, mode mgo.Mode) {
	mc.connectionStr = connectionStr
	s, err := mgo.Dial(mc.connectionStr)
	if err != nil {
		log.Fatalf("Failed to initializing MongoDB instance, error: %s", err.Error())
	}
	mc.rootSession = s
	mc.rootSession.SetMode(mode, true)
	mc.strongSession = s
	mc.strongSession.SetMode(mgo.Strong, true)
}

func (mc *MongoDBStorageDriver) NewSession() *mgo.Session {
	return mc.rootSession.Copy()
}

func (mc *MongoDBStorageDriver) NewStrongSession() *mgo.Session {
	return mc.strongSession.Copy()
}

func (sd *MongoDBStorageDriver) Initialize(args map[string]string) error {
	connStr := args["DRIVER_CONNECTION_STR"]
	if connStr == "" {
		return errors.New("ENV: \"DRIVER_CONNECTION_STR\" is required for initializing storage driver.")
	}
	c, err := mgo.Dial(connStr)
	if err != nil {
		return err
	}
	sd.rootSession = c
	return nil
}

func (sd *MongoDBStorageDriver) SaveCluster(cluster *entities.Cluster, certsMap *certs.GeneratedCertsMap) error {
	session := sd.NewSession()
	defer session.Close()
	db := session.DB("lightning_monkey")
	err := db.C("clusters").Insert(cluster)
	if err != nil {
		return fmt.Errorf("Failed to save cluster to database, error: %s", err.Error())
	}
	certs := []interface{}{}
	if certsMap != nil && certsMap.GetResources() != nil && len(certsMap.GetResources()) > 0 {
		for name, ct := range certsMap.GetResources() {
			certId := bson.NewObjectId()
			certs = append(certs, &entities.Certificate{
				Id:         &certId,
				ClusterId:  cluster.Id,
				Name:       name,
				Content:    ct,
				CreateTime: time.Now(),
			})
		}
	}
	if len(certs) <= 0 {
		return nil
	}
	err = db.C("certificates").Insert(certs...)
	if err != nil {
		return fmt.Errorf("Failed to save cluster to database, error: %s", err.Error())
	}
	return nil
}
