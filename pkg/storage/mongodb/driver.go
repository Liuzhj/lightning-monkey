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
	sd.connectionStr = connStr
	c, err := mgo.Dial(connStr)
	if err != nil {
		return err
	}
	sd.rootSession = c
	return nil
}

func (sd *MongoDBStorageDriver) GetCluster(clusterId string) (*entities.Cluster, error) {
	session := sd.NewSession()
	defer session.Close()
	cId := bson.ObjectIdHex(clusterId)
	var cluster entities.Cluster
	db := session.DB("lightning_monkey")
	err := db.C("clusters").Find(bson.M{"_id": cId}).One(&cluster)
	return &cluster, err
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

func (sd *MongoDBStorageDriver) UpdateCluster(cluster *entities.Cluster) error {
	session := sd.NewSession()
	defer session.Close()
	db := session.DB("lightning_monkey")
	err := db.C("clusters").UpdateId(cluster.Id, cluster)
	if err != nil {
		return fmt.Errorf("Failed to save cluster to database, error: %s", err.Error())
	}
	return nil
}

func (sd *MongoDBStorageDriver) SaveCertificateToCluster(cluster *entities.Cluster, certsMap *certs.GeneratedCertsMap) error {
	session := sd.NewSession()
	defer session.Close()
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
	err := session.DB("lightning_monkey").C("certificates").Insert(certs...)
	if err != nil {
		return fmt.Errorf("Failed to save cluster to database, error: %s", err.Error())
	}
	return nil
}

func (sd *MongoDBStorageDriver) GetCertificatesByClusterId(clusterId string) (entities.CertificateCollection, error) {
	session := sd.NewSession()
	defer session.Close()
	c := session.DB("lightning_monkey").C("certificates")
	cId := bson.ObjectIdHex(clusterId)
	var certs []*entities.Certificate
	err := c.Find(bson.M{"cluster_id": &cId}).All(&certs)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve cluster certificates data, error: %s", err.Error())
	}
	return entities.CertificateCollection(certs), nil
}

func (sd *MongoDBStorageDriver) GetAllClusters() ([]*entities.Cluster, error) {
	session := sd.NewSession()
	defer session.Close()
	var clusters []*entities.Cluster
	err := session.DB("lightning_monkey").C("clusters").Find(nil).All(&clusters)
	return clusters, err
}

func (sd *MongoDBStorageDriver) GetAgentByMetadataId(metadataId string) (*entities.Agent, error) {
	session := sd.NewSession()
	defer session.Close()
	var agent *entities.Agent
	err := session.DB("lightning_monkey").C("agents").Find(bson.M{"metadata_id": metadataId}).One(&agent)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return agent, nil
}

func (sd *MongoDBStorageDriver) GetAllAgentsByClusterId(clusterId string) ([]*entities.Agent, error) {
	session := sd.NewSession()
	defer session.Close()
	cId := bson.ObjectIdHex(clusterId)
	var agents []*entities.Agent
	err := session.DB("lightning_monkey").C("agents").Find(bson.M{"cluster_id": &cId}).All(&agents)
	return agents, err
}

func (sd *MongoDBStorageDriver) SaveAgent(agent *entities.Agent) error {
	session := sd.NewSession()
	defer session.Close()
	_, err := session.DB("lightning_monkey").C("agents").UpsertId(agent.Id, agent)
	return err
}

func (sd *MongoDBStorageDriver) UpdateAgentStatus(agent *entities.Agent) error {
	session := sd.NewSession()
	defer session.Close()
	//update: {"$set": {"some_key.param2": "val2_new", "some_key.param3": "val3_new"}}
	return session.DB("lightning_monkey").C("agents").UpdateId(agent.Id, bson.M{
		"$set": bson.M{
			"last_report_ip":     agent.LastReportIP,
			"last_report_status": agent.LastReportStatus,
			"last_report_time":   agent.LastReportTime,
			"reason":             agent.Reason,
		}})
}

func (sd *MongoDBStorageDriver) BatchUpdateAgentStatus(agents []*entities.Agent) error {
	session := sd.NewSession()
	defer session.Close()
	//update: {"$set": {"some_key.param2": "val2_new", "some_key.param3": "val3_new"}}
	bulk := session.DB("lightning_monkey").C("agents").Bulk()
	for i := 0; i < len(agents); i++ {
		a := agents[i]
		selector := bson.M{"_id": a.Id}
		bulk.Upsert(selector, a)
	}
	_, err := bulk.Run()
	return err
}

func (sd *MongoDBStorageDriver) GetCertificatesByClusterIdAndName(clusterId string, name string) (*entities.Certificate, error) {
	return nil, nil
}
