package model

import (
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
)

func isManyRelation(relationType string) bool {
	return relationType == "hasMany" || relationType == "hasManyThrough" || relationType == "hasAndBelongsToMany"
}

func isSingleRelation(relationType string) bool {
	return relationType == "hasOne" || relationType == "belongsTo"
}

func (loadedModel *Model) ExtractLookupsFromFilter(filterMap *wst.Filter, disableTypeConversions bool) *wst.A {

	if filterMap == nil {
		return nil
	}

	var targetWhere *wst.Where
	if filterMap != nil && filterMap.Where != nil {
		whereCopy := *filterMap.Where
		targetWhere = &whereCopy
	} else {
		targetWhere = nil
	}

	var targetOrder *wst.Order
	if filterMap != nil && filterMap.Order != nil {
		orderValue := *filterMap.Order
		targetOrder = &orderValue
	} else {
		targetOrder = nil
	}
	var targetSkip = filterMap.Skip
	var targetLimit = filterMap.Limit

	var lookups *wst.A
	if targetWhere != nil {
		if !disableTypeConversions {
			datasource.ReplaceObjectIds(*targetWhere)
		}
		lookups = &wst.A{
			{"$match": *targetWhere},
		}
	} else {
		lookups = &wst.A{}
	}

	if targetOrder != nil && len(*targetOrder) > 0 {
		orderMap := bson.D{}
		for _, orderPair := range *targetOrder {
			splt := strings.Split(orderPair, " ")
			key := splt[0]
			directionSt := splt[1]
			if strings.ToLower(strings.TrimSpace(directionSt)) == "asc" {
				//orderMap[key] = 1
				orderMap = append(orderMap, bson.E{Key: key, Value: 1})
			} else if strings.ToLower(strings.TrimSpace(directionSt)) == "desc" {
				//orderMap[key] = -1
				orderMap = append(orderMap, bson.E{Key: key, Value: -1})
			} else {
				panic(fmt.Sprintf("Invalid direction %v while trying to sort by %v", directionSt, key))
			}
		}
		*lookups = append(*lookups, wst.M{
			"$sort": orderMap,
		})
	}

	if targetSkip > 0 {
		*lookups = append(*lookups, wst.M{
			"$skip": targetSkip,
		})
	}
	if targetLimit > 0 {
		*lookups = append(*lookups, wst.M{
			"$limit": targetLimit,
		})
	}

	var targetInclude *wst.Include
	if filterMap != nil && filterMap.Include != nil {
		includeAsInterfaces := *filterMap.Include
		targetInclude = &includeAsInterfaces
	} else {
		targetInclude = nil
	}
	if targetInclude != nil {
		for _, includeItem := range *targetInclude {

			var targetScope *wst.Filter
			if includeItem.Scope != nil {
				scopeValue := *includeItem.Scope
				targetScope = &scopeValue
			} else {
				targetScope = nil
			}

			relationName := includeItem.Relation
			relation := (*loadedModel.Config.Relations)[relationName]
			relatedModelName := relation.Model
			relatedLoadedModel := (*loadedModel.modelRegistry)[relatedModelName]

			if relatedLoadedModel == nil {
				log.Println()
				log.Printf("WARNING: related model %v not found for relation %v.%v", relatedModelName, loadedModel.Name, relationName)
				log.Println()
				continue
			}

			if relatedLoadedModel.Datasource.Name == loadedModel.Datasource.Name {
				switch relation.Type {
				case "belongsTo", "hasOne", "hasMany":
					var matching wst.M
					var lookupLet wst.M
					switch relation.Type {
					case "belongsTo":
						lookupLet = wst.M{
							*relation.ForeignKey: fmt.Sprintf("$%v", *relation.ForeignKey),
						}
						matching = wst.M{
							"$eq": []string{fmt.Sprintf("$%v", *relation.PrimaryKey), fmt.Sprintf("$$%v", *relation.ForeignKey)},
						}
						break
					case "hasOne", "hasMany":
						lookupLet = wst.M{
							*relation.ForeignKey: fmt.Sprintf("$%v", *relation.PrimaryKey),
						}
						matching = wst.M{
							"$eq": []string{fmt.Sprintf("$%v", *relation.ForeignKey), fmt.Sprintf("$$%v", *relation.ForeignKey)},
						}
						break
					}
					pipeline := []interface{}{
						wst.M{
							"$match": wst.M{
								"$expr": wst.M{
									"$and": wst.A{
										matching,
									},
								},
							},
						},
					}
					project := wst.M{}
					for _, propertyName := range relatedLoadedModel.Config.Hidden {
						project[propertyName] = false
					}
					if len(project) > 0 {
						pipeline = append(pipeline, wst.M{
							"$project": project,
						})
					}
					if targetScope != nil {
						nestedLoopkups := relatedLoadedModel.ExtractLookupsFromFilter(targetScope, disableTypeConversions)
						if nestedLoopkups != nil {
							for _, v := range *nestedLoopkups {
								pipeline = append(pipeline, v)
							}
						}
					}

					*lookups = append(*lookups, wst.M{
						"$lookup": wst.M{
							"from":     relatedLoadedModel.CollectionName,
							"let":      lookupLet,
							"pipeline": pipeline,
							"as":       relationName,
						},
					})
					break
				}
				switch relation.Type {
				case "hasOne", "belongsTo":
					*lookups = append(*lookups, wst.M{
						"$unwind": wst.M{
							"path":                       fmt.Sprintf("$%v", relationName),
							"preserveNullAndEmptyArrays": true,
						},
					})
					break
				}

			}
		}

	} else {

	}
	return lookups
}

/*
params:
	- relationDeepLevel: Starts at 1 (Root is 0)
*/
func (loadedModel *Model) mergeRelated(relationDeepLevel byte, documents *wst.A, includeItem wst.IncludeItem, baseContext *EventContext) error {

	if documents == nil {
		return nil
	}

	parentDocs := documents

	relationName := includeItem.Relation
	relation := (*loadedModel.Config.Relations)[relationName]
	relatedModelName := relation.Model
	relatedLoadedModel := (*loadedModel.modelRegistry)[relatedModelName]

	parentModel := loadedModel
	parentRelationName := relationName

	if relatedLoadedModel == nil {
		log.Println()
		log.Printf("WARNING: related model %v not found for relation %v.%v", relatedModelName, loadedModel.Name, relationName)
		log.Println()
		return nil
	}

	//if relation.Options.SkipAuth && relationDeepLevel > 1 {
	// Only skip auth checking for relations above the level 1
	if relation.Options.SkipAuth {
		if loadedModel.App.Debug {
			log.Printf("DEBUG: SkipAuth %v.%v\n", loadedModel.Name, relationName)
		}
	} else {
		objId := "*"
		if len(*documents) == 1 {
			objId = (*documents)[0]["_id"].(primitive.ObjectID).Hex()
		}

		action := fmt.Sprintf("__get__%v", relationName)
		if loadedModel.App.Debug {
			log.Printf("DEBUG: Check %v.%v\n", loadedModel.Name, action)
		}
		err, allowed := loadedModel.EnforceEx(baseContext.Bearer, objId, action, baseContext)
		if err != nil && err != fiber.ErrUnauthorized {
			return err
		}
		if !allowed {
			for _, doc := range *documents {
				delete(doc, relationName)
			}
		}
	}

	if relatedLoadedModel.Datasource.Name != loadedModel.Datasource.Name {
		switch relation.Type {
		case "belongsTo", "hasOne", "hasMany":
			keyFrom := ""
			keyTo := ""
			switch relation.Type {
			case "belongsTo":
				keyFrom = *relation.PrimaryKey
				keyTo = *relation.ForeignKey
				break
			case "hasOne", "hasMany":
				keyFrom = *relation.ForeignKey
				keyTo = *relation.PrimaryKey
				break
			}

			var targetScope *wst.Filter
			if includeItem.Scope != nil {
				scopeValue := *includeItem.Scope
				targetScope = &scopeValue
			} else {
				targetScope = &wst.Filter{}
			}

			wasEmptyWhere := false
			if targetScope.Where == nil {
				targetScope.Where = &wst.Where{}
				wasEmptyWhere = true
			}

			cachedRelatedDocs := make([]InstanceA, len(*documents))
			localCache := map[string]InstanceA{}

			for documentIdx, document := range *documents {

				if wasEmptyWhere && relatedLoadedModel.Config.Cache.Datasource != "" /* && keyFrom == relatedLoadedModel.Config.Cache.Keys*/ {

					cacheDs, err := loadedModel.App.FindDatasource(relatedLoadedModel.Config.Cache.Datasource)
					if err != nil {
						return err
					}
					safeCacheDs := cacheDs.(*datasource.Datasource)

					//baseKey := fmt.Sprintf("%v:%v", safeCacheDs.Viper.GetString(safeCacheDs.Key+".database"), relatedLoadedModel.Config.Name)
					for _, keyGroup := range relatedLoadedModel.Config.Cache.Keys {

						if len(keyGroup) == 1 && keyGroup[0] == keyFrom {

							cacheKeyTo := fmt.Sprintf("%v:%v", keyFrom, document[keyTo])

							if localCache[cacheKeyTo] != nil {
								cachedRelatedDocs[documentIdx] = localCache[cacheKeyTo]
							} else {
								var cachedDocs *wst.A

								cacheLookups := &wst.A{wst.M{"$match": wst.M{keyFrom: cacheKeyTo}}}
								cachedDocs, err = safeCacheDs.FindMany(relatedLoadedModel.CollectionName, cacheLookups)
								if err != nil {
									return err
								}

								for _, cachedDoc := range *cachedDocs {
									cachedInstance := relatedLoadedModel.Build(cachedDoc, baseContext)
									if cachedRelatedDocs[documentIdx] == nil {
										cachedRelatedDocs[documentIdx] = InstanceA{}
									}
									cachedRelatedDocs[documentIdx] = append(cachedRelatedDocs[documentIdx], cachedInstance)
								}
								localCache[cacheKeyTo] = cachedRelatedDocs[documentIdx]
							}

						}
					}

				}

				relatedInstances := cachedRelatedDocs[documentIdx]
				if relatedInstances == nil {

					(*targetScope.Where)[keyFrom] = document[keyTo]
					if isSingleRelation(relation.Type) {
						targetScope.Limit = 1
					}

					var err error
					relatedInstances, err = relatedLoadedModel.FindMany(targetScope, baseContext)
					if err != nil {
						return err
					}
				} else {
					if loadedModel.App.Debug {
						log.Printf("Found cache for %v.%v[%v]\n", loadedModel.Name, relationName, documentIdx)
					}
				}

				if loadedModel.hasHiddenProperties {
					for _, relatedInstance := range relatedInstances {
						relatedInstance.HideProperties()
					}
				}

				switch {
				case isSingleRelation(relation.Type):
					if len(relatedInstances) > 0 {
						document[relationName] = relatedInstances[0]
					} else {
						document[relationName] = nil
					}
					break
				case isManyRelation(relation.Type):
					document[relationName] = relatedInstances
					break
				}

			}

			break
		}

	} else {

		if includeItem.Scope != nil && documents != nil && len(*documents) > 0 {
			if includeItem.Scope.Include != nil {

				for _, includeItem := range *includeItem.Scope.Include {
					relationName := includeItem.Relation
					//relation := (*loadedModel.Config.Relations)[relationName]
					_isSingleRelation := isSingleRelation(relation.Type)
					_isManyRelation := !_isSingleRelation
					//relatedModelName := relation.Model
					//relatedLoadedModel := (*loadedModel.modelRegistry)[relatedModelName]

					nestedDocuments := make(wst.A, 0)

					for _, doc := range *documents {

						switch {
						case _isSingleRelation:
							if doc[parentRelationName] != nil {
								documentsValue := make(wst.A, 1)

								if relatedInstance, ok := doc[parentRelationName].(map[string]interface{}); ok {
									documentsValue[0] = wst.M{}
									for k, v := range relatedInstance {
										documentsValue[0][k] = v
									}
								} else if relatedInstance, ok := doc[parentRelationName].(wst.M); ok {
									documentsValue[0] = relatedInstance
								} else {
									log.Printf("WARNING: Invalid type for %v.%v %s\n", loadedModel.Name, relationName, doc[parentRelationName])
								}

								//documents = &documentsValue
								nestedDocuments = append(nestedDocuments, documentsValue...)
							}
							break
						case _isManyRelation:
							if doc[parentRelationName] != nil {

								if asGeneric, ok := doc[parentRelationName].([]interface{}); ok {
									relatedInstances := asGeneric
									nestedDocuments = append(nestedDocuments, *wst.AFromGenericSlice(&relatedInstances)...)
								} else if asPrimitiveA, ok := doc[parentRelationName].(primitive.A); ok {
									relatedInstances := asPrimitiveA
									nestedDocuments = append(nestedDocuments, *wst.AFromPrimitiveSlice(&relatedInstances)...)
								} else if asA, ok := doc[parentRelationName].(wst.A); ok {
									nestedDocuments = append(nestedDocuments, asA...)
								} else {
									log.Println("WARNING: unknown type for relation", relationName, "in", loadedModel.Name)
									continue
								}

							}
							break
						}

					}
					loadedModel := relatedLoadedModel
					if loadedModel.App.Debug {
						log.Printf("Dispatch nested relation %v.%v.%v (n=%v, m=%v)\n", parentModel.Name, parentRelationName, relationName, len(*parentDocs), len(nestedDocuments))
					}
					err := loadedModel.mergeRelated(relationDeepLevel+1, &nestedDocuments, includeItem, baseContext)
					if err != nil {
						return err
					}

				}
			}
		}
	}

	return nil
}
