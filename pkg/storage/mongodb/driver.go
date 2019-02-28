package mongodb

import (
	"context"
	"errors"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/certs"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/globalsign/mgo/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type MongoDBStorageDriver struct {
	client *mongo.Client
}

func (sd *MongoDBStorageDriver) Initialize(args map[string]string) error {
	connStr := args["DRIVER_CONNECTION_STR"]
	if connStr == "" {
		return errors.New("ENV: \"DRIVER_CONNECTION_STR\" is required for initializing storage driver.")
	}
	c, err := mongo.NewClient(options.Client().ApplyURI(connStr))
	if err != nil {
		return err
	}
	err = c.Connect(context.Background())
	if err != nil {
		return err
	}
	sd.client = c
	return nil
}

func (sd *MongoDBStorageDriver) SaveCluster(cluster *entities.Cluster, certsMap *certs.GeneratedCertsMap) error {
	db := sd.client.Database("lightning_monkey")
	defer db.Client().Disconnect(context.Background())

	c := db.Collection("clusters")
	err := db.Client().UseSession(context.Background(), func(sessionCxt mongo.SessionContext) error {
		err := sessionCxt.StartTransaction()
		if err != nil {
			return err
		}
		_, err = c.InsertOne(sessionCxt, cluster)
		if err != nil {
			return err
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
		_, err = db.Collection("certificates").InsertMany(sessionCxt, certs)
		if err != nil {
			return sessionCxt.AbortTransaction(sessionCxt)
		}
		return sessionCxt.CommitTransaction(sessionCxt)
	})
	if err != nil {
		return fmt.Errorf("Failed to save cluster to database, error: %s", err.Error())
	}
	return nil
}
