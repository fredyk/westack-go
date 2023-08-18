package model

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/datasource"
)

var AllowedStages = []string{
	"$addFields",
	"$group",
	"$project",
	"$search",
	"$set",
	"$unset",
	"$unwind",
}

func isManyRelation(relationType string) bool {
	return relationType == "hasMany" || relationType == "hasManyThrough" || relationType == "hasAndBelongsToMany"
}

func isSingleRelation(relationType string) bool {
	return relationType == "hasOne" || relationType == "belongsTo"
}

func (loadedModel *Model) ExtractLookupsFromFilter(filterMap *wst.Filter, disableTypeConversions bool) (*wst.A, error) {

	if filterMap == nil {
		return nil, nil
	}

	var targetWhere *wst.Where
	if filterMap != nil && filterMap.Where != nil {
		whereCopy := *filterMap.Where
		targetWhere = &whereCopy
	} else {
		targetWhere = nil
	}

	var targetAggregationBeforeLookups []wst.AggregationStage
	var targetAggregationAfterLookups []wst.AggregationStage
	var newFoundFields = make(map[string]bool)
	if filterMap != nil && filterMap.Aggregation != nil {
		for _, aggregationStage := range filterMap.Aggregation {
			var validStageFound = false
			var firstKeyFound = ""
			for key, _ := range aggregationStage {
				firstKeyFound = key
				if !validStageFound {
					for _, allowedStage := range AllowedStages {
						if key == allowedStage {
							validStageFound = true
							break
						}
					}
				}
				switch key {
				case "$addFields", "$project", "$set":
					fieldCount := 0
					// Check wether it affects a nested relation or not
					for fieldName, fieldValue := range aggregationStage[key].(map[string]interface{}) {
						var placeToInsert string // "BEFORE" or "AFTER"
						switch fieldValue.(type) {
						case string:
							if strings.Contains(fieldValue.(string), ".") {
								// Nested relation
								placeToInsert = "AFTER"
								// In adition, extract the first part of the "foo.bar", and check if foo is a valid relation of the loadedModel
								parts := strings.Split(fieldValue.(string), ".")
								relationName := strings.ReplaceAll(parts[0], "$", "")
								if relation, ok := (*loadedModel.Config.Relations)[relationName]; !ok {
									return nil, wst.CreateError(fiber.ErrBadRequest,
										"BAD_RELATION",
										fiber.Map{"message": fmt.Sprintf("relation %v not found for model %v", relationName, loadedModel.Name)},
										"ValidationError",
									)
								} else {
									// ensure that the relation is in the same datasource

									relatedModel, err := loadedModel.App.FindModel(relation.Model)
									if err != nil {
										return nil, err
									}

									if relatedModel.(*Model).Datasource.Name != loadedModel.Datasource.Name {
										return nil, wst.CreateError(fiber.ErrBadRequest,
											"BAD_RELATION",
											fiber.Map{"message": fmt.Sprintf("related model %v at relation %v belongs to another datasource", relatedModel.(*Model).Name, relationName)},
											"ValidationError",
										)
									}
								}

							} else {
								// Not nested relation
								placeToInsert = "BEFORE"
							}
						default:
							// Not nested relation
							placeToInsert = "BEFORE"
						}
						if fieldCount == 0 {
							switch placeToInsert {
							case "BEFORE":
								targetAggregationBeforeLookups = append(targetAggregationBeforeLookups, wst.AggregationStage{})
							case "AFTER":
								newFoundFields[fieldName] = true
								targetAggregationAfterLookups = append(targetAggregationAfterLookups, wst.AggregationStage{})
							}
						}
						switch placeToInsert {
						case "BEFORE":
							targetAggregationBeforeLookups[len(targetAggregationBeforeLookups)-1][key] = aggregationStage[key]
						case "AFTER":
							targetAggregationAfterLookups[len(targetAggregationAfterLookups)-1][key] = aggregationStage[key]
						}
						fieldCount++
					}
				}
			}
			if !validStageFound {
				return nil, fmt.Errorf("%s aggregation stage not allowed", firstKeyFound)
			}
		}
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

	var lookups *wst.A = &wst.A{}
	for _, aggregationStage := range targetAggregationBeforeLookups {
		*lookups = append(*lookups, wst.CopyMap(wst.M(aggregationStage)))
	}
	var targetMatchBeforeLookups wst.M
	var targetMatchAfterLookups wst.M
	if targetWhere != nil {
		targetWhereAsM := wst.M(*targetWhere)
		if !disableTypeConversions {
			datasource.ReplaceObjectIds(*targetWhere)
		}
		targetMatchBeforeLookups = wst.M{"$match": recursiveExtractFields(targetWhereAsM, newFoundFields, "EXCLUDE")}
		*lookups = append(*lookups, targetMatchBeforeLookups)
		aux := recursiveExtractFields(targetWhereAsM, newFoundFields, "INCLUDE")
		if len(aux) > 0 {
			targetMatchAfterLookups = wst.M{"$match": aux}
		}
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
				return nil, fmt.Errorf("invalid direction %v while trying to sort by %v", directionSt, key)
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
			if relation == nil {
				return nil, fmt.Errorf("warning: relation %v not found for model %v", relationName, loadedModel.Name)
			}

			relatedModelName := relation.Model
			relatedLoadedModel := (*loadedModel.modelRegistry)[relatedModelName]

			if relatedLoadedModel == nil {
				return nil, fmt.Errorf("warning: related model %v not found for relation %v.%v", relatedModelName, loadedModel.Name, relationName)
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
					pipeline := wst.A{
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
						nestedLoopkups, err := relatedLoadedModel.ExtractLookupsFromFilter(targetScope, disableTypeConversions)
						if err != nil {
							return nil, err
						}
						if nestedLoopkups != nil {
							for _, v := range *nestedLoopkups {
								pipeline = append(pipeline, v)
							}
						}
					}

					// limit "belongsTo" and "hasOne" to 2 documents, in order to check later if there is more than one
					if relation.Type == "belongsTo" || relation.Type == "hasOne" {
						pipeline = append(pipeline, wst.M{
							"$limit": 2,
						})
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

	}
	for _, aggregationStage := range targetAggregationAfterLookups {
		*lookups = append(*lookups, wst.CopyMap(wst.M(aggregationStage)))
	}
	if len(targetMatchAfterLookups) > 0 {
		*lookups = append(*lookups, targetMatchAfterLookups)
	}
	if loadedModel.App.Debug {
		marshalled, err := json.MarshalIndent(lookups, "", "  ")
		if err != nil {
			return nil, err
		}
		log.Printf("DEBUG: lookups %v\n", string(marshalled))
	}

	return lookups, nil
}

func recursiveExtractFields(targetWhere wst.M, fieldsToExclude map[string]bool, mode string) wst.M {
	result := wst.M{}
	// Some posible wheres:
	// targetWhere = {"foo": "bar"}
	// targetWhere = {"$and": [{"foo1": "bar1"}, {"foo2": "bar2"}]}
	// targetWhere = {"foo": {$exists: true}}
	// and lots of other mongo expressions
	// mode: "INCLUDE" || "EXCLUDE"
	for key, value := range targetWhere {
		switch key {
		case "$and":
			newAnd := recursiveExtractExpression(key, value, fieldsToExclude, mode)
			if len(newAnd) > 0 {
				result["$and"] = newAnd
			}
		case "$or":
			newOr := recursiveExtractExpression(key, value, fieldsToExclude, mode)
			if len(newOr) > 0 {
				result["$or"] = newOr
			}
		default:
			/*if !fieldsToExclude[key] {
				result[key] = value
			}*/
			switch mode {
			case "INCLUDE":
				if fieldsToExclude[key] {
					result[key] = value
				}
			case "EXCLUDE":
				if !fieldsToExclude[key] {
					result[key] = value
				}
			}
		}
	}
	return result
}

func recursiveExtractExpression(key string, value interface{}, fieldsToExclude map[string]bool, mode string) []interface{} {
	newList := make([]interface{}, 0)
	var asInterfaceList []interface{}
	switch value.(type) {
	case []interface{}:
		asInterfaceList = value.([]interface{})
	case []wst.M:
		for _, v := range value.([]wst.M) {
			asInterfaceList = append(asInterfaceList, v)
		}
	case []map[string]interface{}:
		for _, v := range value.([]map[string]interface{}) {
			asInterfaceList = append(asInterfaceList, v)
		}
	}
	for _, andValue := range asInterfaceList {
		var asM wst.M
		if v, ok := andValue.(wst.M); ok {
			asM = v
		} else if v, ok = andValue.(map[string]interface{}); ok {
			asM = make(wst.M, 0)
			for k, v := range v {
				asM[k] = v
			}
		}
		newVal := recursiveExtractFields(asM, fieldsToExclude, mode)
		if len(newVal) > 0 {
			newList = append(newList, newVal)
		}
	}
	return newList
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

			disabledCache := loadedModel.App.Viper.GetBool("disableCache")
			for documentIdx, document := range *documents {

				if !disabledCache && wasEmptyWhere && relatedLoadedModel.Config.Cache.Datasource != "" /* && keyFrom == relatedLoadedModel.Config.Cache.Keys*/ {

					cacheDs, err := loadedModel.App.FindDatasource(relatedLoadedModel.Config.Cache.Datasource)
					if err != nil {
						return err
					}
					safeCacheDs := cacheDs.(*datasource.Datasource)

					//baseKey := fmt.Sprintf("%v:%v", safeCacheDs.Viper.GetString(safeCacheDs.Key+".database"), relatedLoadedModel.Config.Name)
					for _, keyGroup := range relatedLoadedModel.Config.Cache.Keys {

						if len(keyGroup) == 1 && keyGroup[0] == keyFrom {

							var documentKeyTo = document[keyTo]
							switch documentKeyTo.(type) {
							case primitive.ObjectID:
								documentKeyTo = documentKeyTo.(primitive.ObjectID).Hex()
							}
							var includePrefix = ""
							if targetScope.Include != nil {
								marshalledTargetInclude, err := json.Marshal(targetScope.Include)
								if err != nil {
									return err
								}
								includePrefix = fmt.Sprintf("_inc_%s_", marshalledTargetInclude)
							}
							if targetScope.Where == nil {
								targetScope.Where = &wst.Where{}
							}
							(*targetScope.Where)[keyFrom] = documentKeyTo
							marshalledTargetWhere, err := json.Marshal(targetScope.Where)
							if err != nil {
								return err
							}
							includePrefix += fmt.Sprintf("_whr_%s_", marshalledTargetWhere)
							cacheKeyTo := fmt.Sprintf("%v%v:%v", includePrefix, keyFrom, documentKeyTo)

							if localCache[cacheKeyTo] != nil {
								cachedRelatedDocs[documentIdx] = localCache[cacheKeyTo]
							} else {
								var cachedDocs []wst.M

								cacheLookups := &wst.A{wst.M{"$match": wst.M{keyFrom: cacheKeyTo}}}
								if loadedModel.App.Debug {
									log.Printf("DEBUG: cacheLookups %v\n", cacheLookups)
								}
								cursor, err := safeCacheDs.FindMany(relatedLoadedModel.CollectionName, cacheLookups)
								if err != nil {
									return err
								}
								err = cursor.All(context.Background(), &cachedDocs)
								if err != nil {
									cursor.Close(context.Background())
									return err
								}
								cursor.Close(context.Background())

								nestedDocsCache := NewBuildCache()
								for _, cachedDoc := range cachedDocs {
									cachedInstance, err := relatedLoadedModel.Build(cachedDoc, nestedDocsCache, baseContext)
									if err != nil {
										return err
									}
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
					relatedInstances, err = relatedLoadedModel.FindMany(targetScope, baseContext).All()
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
