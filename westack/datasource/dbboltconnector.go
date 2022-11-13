package datasource

import (
	"github.com/boltdb/bolt"
	"go.mongodb.org/mongo-driver/bson"

	wst "github.com/fredyk/westack-go/westack/common"
)

func DbBoltFindMany(db *bolt.DB, mongoPipeline *wst.A) (*wst.A, error) {

	results := &wst.A{}

	err := EmulateMongoAggregation(db, mongoPipeline, results)

	return results, err
}

func EmulateMongoAggregation(db *bolt.DB, pipeline *wst.A, output *wst.A) error {

	for _, stage := range *pipeline {

		stageMap := stage
		for stageName, stageValue := range stageMap {
			switch stageName {
			case "$match":
				err := EmulateMongoMatchStage(db, stageValue, output)
				if err != nil {
					return err
				}
			case "$lookup":
				err := EmulateMongoLookupStage(db, stageValue, output)
				if err != nil {
					return err
				}
			case "$project":
				err := EmulateMongoProjectStage(db, stageValue, output)
				if err != nil {
					return err
				}
			case "$unwind":
				err := EmulateMongoUnwindStage(db, stageValue, output)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func EmulateMongoMatchStage(db *bolt.DB, matchSpec interface{}, output *wst.A) interface{} {

	var normalizedMatchSpec bson.A

	// Normalize matchSpec to map[string]interface{}
	switch matchSpec.(type) {
	case bson.A:
		normalizedMatchSpec = matchSpec.(bson.A)
	case bson.M:
		normalizedMatchSpec = bson.A{matchSpec}

	}

	tx, err := db.Begin(false)
	if err != nil {
		return err
	}

	//defer tx.Rollback()

	bucket := tx.Bucket([]byte("main"))

	c := bucket.Cursor()

	for k, v := c.First(); k != nil; k, v = c.Next() {

	}

	return nil
}
