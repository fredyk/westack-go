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

	wst "github.com/fredyk/westack-go/v3/common"
	"github.com/fredyk/westack-go/v3/datasource"
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

func (loadedModel *StatefulModel) ExtractLookupsFromFilter(filterMap *wst.Filter, disableTypeConversions bool) (*wst.A, error) {

	if filterMap == nil {
		return nil, nil
	}

	var targetWhere *wst.Where
	if filterMap.Where != nil {
		whereCopy := *filterMap.Where
		targetWhere = &whereCopy
	} else {
		targetWhere = nil
	}

	// Fields and ExcludeFields are exclusive
	if filterMap.Fields != nil && filterMap.ExcludeFields != nil {
		if len(*filterMap.Fields) > 0 && len(*filterMap.ExcludeFields) > 0 {
			return nil, wst.CreateError(fiber.ErrBadRequest, "FIELDS_EXCLUDE_FIELDS_CONFLICT", nil, "ValidationError")
		}
	}

	var targetProjectIncludeFields wst.Fields
	if filterMap.Fields != nil && len(*filterMap.Fields) > 0 {
		fieldsCopy := *filterMap.Fields
		targetProjectIncludeFields = fieldsCopy
	} else {
		targetProjectIncludeFields = nil
	}

	var targetProjectExcludeFields wst.Fields
	if filterMap.ExcludeFields != nil && len(*filterMap.ExcludeFields) > 0 {
		excludeFieldsCopy := *filterMap.ExcludeFields
		targetProjectExcludeFields = excludeFieldsCopy
	} else {
		targetProjectExcludeFields = nil
	}

	var targetAggregationBeforeLookups []wst.AggregationStage
	var targetAggregationAfterLookups []wst.AggregationStage
	var newFoundFields = make(map[string]bool)
	if filterMap.Aggregation != nil {
		for _, aggregationStage := range filterMap.Aggregation {
			var validStageFound = false
			var firstKeyFound = ""
			for key := range aggregationStage {
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
					fieldCountForBefore := 0
					fieldCountForAfter := 0
					// Check wether it affects a nested relation or not
					for fieldName, fieldValue := range aggregationStage[key].(map[string]interface{}) {
						var placeToInsert string // "BEFORE" or "AFTER"
						switch fieldValue.(type) {
						case string:
							if strings.Contains(fieldValue.(string), ".") {
								// Nested relation
								placeToInsert = "AFTER"
								// Input adition, extract the first part of the "foo.bar", and check if foo is a valid relation of the loadedModel
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

									relatedModel, _ := loadedModel.App.FindModel(relation.Model)

									if relatedModel.Datasource.Name != loadedModel.Datasource.Name {
										return nil, wst.CreateError(fiber.ErrBadRequest,
											"BAD_RELATION",
											fiber.Map{"message": fmt.Sprintf("related model %v at relation %v belongs to another datasource", relatedModel.Name, relationName)},
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
						switch placeToInsert {
						case "BEFORE":
							if fieldCountForBefore == 0 {
								targetAggregationBeforeLookups = append(targetAggregationBeforeLookups, wst.AggregationStage{})
							}
							targetAggregationBeforeLookups[len(targetAggregationBeforeLookups)-1][key] = aggregationStage[key]
							fieldCountForBefore++
						case "AFTER":
							if fieldCountForAfter == 0 {
								newFoundFields[fieldName] = true
								targetAggregationAfterLookups = append(targetAggregationAfterLookups, wst.AggregationStage{})
							}
							targetAggregationAfterLookups[len(targetAggregationAfterLookups)-1][key] = aggregationStage[key]
							fieldCountForAfter++
						}
					}
				}
			}
			if !validStageFound {
				//return nil, fmt.Errorf("%s aggregation stage not allowed", firstKeyFound)
				return nil, wst.CreateError(fiber.ErrBadRequest,
					"BAD_AGGREGATION_STAGE",
					fiber.Map{"message": fmt.Sprintf("%s aggregation stage not allowed", firstKeyFound)},
					"ValidationError",
				)
			}
		}
	}

	var targetOrder *wst.Order
	if filterMap.Order != nil {
		orderValue := *filterMap.Order
		targetOrder = &orderValue
	} else {
		targetOrder = nil
	}
	var targetSkip = filterMap.Skip
	var targetLimit = filterMap.Limit

	var lookups = &wst.A{}
	for _, aggregationStage := range targetAggregationBeforeLookups {
		*lookups = append(*lookups, wst.CopyMap(wst.M(aggregationStage)))
	}
	var targetMatchAfterLookups wst.M
	if targetWhere != nil {
		targetWhereAsM := wst.M(*targetWhere)
		if !disableTypeConversions {
			_, err := datasource.ReplaceObjectIds(*targetWhere)
			if err != nil {
				return nil, err
			}
		}
		var extractedMatch wst.M
		newFoundFields, extractedMatch, _ = recursiveExtractFields(targetWhereAsM, newFoundFields, "EXCLUDE")
		if len(extractedMatch) > 0 {
			*lookups = append(*lookups, wst.M{"$match": extractedMatch})
		}
		newFoundFields, extractedMatch, _ = recursiveExtractFields(targetWhereAsM, newFoundFields, "INCLUDE")
		if len(extractedMatch) > 0 {
			targetMatchAfterLookups = wst.M{"$match": extractedMatch}
		}
	}

	const ProjectModeInclude = 1

	// Build projection/unset stages
	var projectFields wst.Fields
	useExclude := false
	if len(targetProjectExcludeFields) > 0 {
		useExclude = true
		projectFields = targetProjectExcludeFields
	} else {
		projectFields = targetProjectIncludeFields
	}

	if useExclude && len(projectFields) > 0 {
		// Prefer $unset for excludes. Handle _id separately with $project: {_id: 0}.
		unsetFields := make([]string, 0, len(projectFields))
		excludeId := false
		for _, fieldName := range projectFields {
			switch fieldName {
			case "id":
				fieldName = "_id"
				excludeId = true
			case "_id":
				excludeId = true
			}
			if fieldName == "_id" {
				continue
			}
			unsetFields = append(unsetFields, fieldName)
		}
		if excludeId {
			*lookups = append(*lookups, wst.M{
				"$project": wst.M{"_id": 0},
			})
		}
		if len(unsetFields) > 0 {
			*lookups = append(*lookups, wst.M{
				"$unset": unsetFields,
			})
		}
	} else if len(projectFields) > 0 {
		fieldsStage := wst.M{}

		// Helper to add nested field projection using path expressions
		var addNested func(target wst.M, segments []string, fullPath string)
		addNested = func(target wst.M, segments []string, fullPath string) {
			if len(segments) == 0 {
				return
			}
			head := segments[0]
			if len(segments) == 1 {
				// If parent already fully included, nothing to do
				if existing, ok := target[head]; ok && existing == ProjectModeInclude {
					return
				}
				target[head] = "$" + fullPath
				return
			}
			if existing, ok := target[head]; ok {
				if existing == ProjectModeInclude {
					// parent fully included; nothing else to do
					return
				}
				if m, ok := existing.(wst.M); ok {
					addNested(m, segments[1:], fullPath)
					return
				}
			}
			child := wst.M{}
			target[head] = child
			addNested(child, segments[1:], fullPath)
		}

		for _, fieldName := range projectFields {
			switch fieldName {
			case "id":
				continue
			case "_id":
				continue
			}
			if strings.Contains(fieldName, ".") {
				segments := strings.Split(fieldName, ".")
				// If top-level already fully included, skip nested projection
				if v, ok := fieldsStage[segments[0]]; ok && v == ProjectModeInclude {
					continue
				}
				addNested(fieldsStage, segments, fieldName)
			} else {
				fieldsStage[fieldName] = ProjectModeInclude
			}
		}
		// Always include _id for include projections (to keep Model.Build() behavior mapping _id -> id)
		fieldsStage["_id"] = 1

		*lookups = append(*lookups, wst.M{
			"$project": fieldsStage,
		})
	}

	var targetOrderBeforeLookups bson.D
	var targetOrderAfterLookups bson.D
	if targetOrder != nil && len(*targetOrder) > 0 {
		for _, orderPair := range *targetOrder {
			splt := strings.Split(orderPair, " ")
			key := splt[0]
			directionSt := splt[1]

			var orderEntry bson.E
			if strings.ToLower(strings.TrimSpace(directionSt)) == "asc" {
				//orderMap[key] = 1
				orderEntry = bson.E{Key: key, Value: 1}
			} else if strings.ToLower(strings.TrimSpace(directionSt)) == "desc" {
				//orderMap[key] = -1
				orderEntry = bson.E{Key: key, Value: -1}
			} else {
				return nil, fmt.Errorf("invalid direction %v while trying to sort by %v", directionSt, key)
			}

			// when the first complex order key was found, all the following keys will be added to the targetOrderAfterLookups
			if !strings.Contains(key, ".") && targetOrderAfterLookups == nil {
				targetOrderBeforeLookups = append(targetOrderBeforeLookups, orderEntry)
			} else {
				// if targetOrderAfterLookups == nil {
				// 	targetOrderAfterLookups = bson.D{}
				// }
				targetOrderAfterLookups = append(targetOrderAfterLookups, orderEntry)
			}

		}
	}

	if len(targetOrderBeforeLookups) > 0 {
		*lookups = append(*lookups, wst.M{
			"$sort": targetOrderBeforeLookups,
		})
	}

	if len(targetMatchAfterLookups) == 0 && len(targetOrderAfterLookups) == 0 {
		// skip and limit before lookups, buf after first match
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
	}

	var targetInclude *wst.Include
	if filterMap.Include != nil {
		includeAsInterfaces := *filterMap.Include
		targetInclude = &includeAsInterfaces
	} else {
		targetInclude = nil
	}
	if targetInclude != nil {
		for _, includeItem := range *targetInclude {
			var err error
			lookups, err = loadedModel.appendIncludeToLookups(includeItem, disableTypeConversions, lookups)
			if err != nil {
				return nil, err
			}
		}

	}
	for _, aggregationStage := range targetAggregationAfterLookups {
		*lookups = append(*lookups, wst.CopyMap(wst.M(aggregationStage)))
	}

	if len(targetMatchAfterLookups) > 0 {
		*lookups = append(*lookups, targetMatchAfterLookups)
	}

	if len(targetMatchAfterLookups) > 0 || len(targetOrderAfterLookups) > 0 {
		if len(targetOrderAfterLookups) > 0 {
			*lookups = append(*lookups, wst.M{
				"$sort": targetOrderAfterLookups,
			})
		}
		// skip and limit after lookups and match
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
	}

	if loadedModel.App.Debug {
		marshalled, err := json.MarshalIndent(lookups, "", "  ")
		if err != nil {
			return nil, err
		}
		log.Printf("[DEBUG] lookups %v\n", string(marshalled))
	}

	return lookups, nil
}

func (loadedModel *StatefulModel) appendIncludeToLookups(includeItem wst.IncludeItem, disableTypeConversions bool, lookups *wst.A) (*wst.A, error) {
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

	relatedModel := relatedLoadedModel
	if relatedModel.Datasource.Name == loadedModel.Datasource.Name {
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
			for _, propertyName := range relatedModel.Config.Hidden {
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
					pipeline = append(pipeline, *nestedLoopkups...)
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
					"from":     relatedModel.CollectionName,
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
	return lookups, nil
}

func recursiveExtractFields(targetWhere wst.M, specialFields map[string]bool, mode string) (outSpecialFields map[string]bool, result wst.M, foundSpecialFields bool) {
	outSpecialFields = make(map[string]bool)
	result = wst.M{}
	// Some posible wheres:
	// targetWhere = {"foo": "bar"}
	// targetWhere = {"$and": [{"foo1": "bar1"}, {"foo2": "bar2"}]}
	// targetWhere = {"foo": {$exists: true}}
	// and lots of other mongo expressions
	// mode: "INCLUDE" || "EXCLUDE"
	// Copy specialFields to avoid modifying the original map
	for key, value := range specialFields {
		outSpecialFields[key] = value
	}
	for key, value := range targetWhere {
		switch key {
		case "$and":
			var newAnd []interface{}
			outSpecialFields, newAnd, foundSpecialFields = recursiveExtractExpression(key, value, outSpecialFields, mode)
			if len(newAnd) > 0 {
				result["$and"] = newAnd
			}
		case "$or":
			//var newOr []interface{}
			outSpecialFields, _, foundSpecialFields = recursiveExtractExpression(key, value, outSpecialFields, mode)
			if len(outSpecialFields) > 0 {
				if mode == "INCLUDE" {
					result["$or"] = value
				}
			} else if !foundSpecialFields && mode == "EXCLUDE" {
				result["$or"] = value
			}
		default:
			// check if key is a nested field
			if strings.Contains(key, ".") {
				foundSpecialFields = true
				outSpecialFields[key] = true
			}
			switch mode {
			case "INCLUDE":
				if outSpecialFields[key] {
					result[key] = value
				}
			case "EXCLUDE":
				if !outSpecialFields[key] {
					result[key] = value
				}
			}
		}
	}
	return
}

func recursiveExtractExpression(key string, value interface{}, specialFields map[string]bool, mode string) (outSpecialFields map[string]bool, newList []interface{}, foundSpecialFields bool) {
	newList = make([]interface{}, 0)
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
		var newVal wst.M
		outSpecialFields, newVal, foundSpecialFields = recursiveExtractFields(asM, specialFields, mode)
		if len(newVal) > 0 {
			newList = append(newList, newVal)
		}
	}
	return
}

/*
params:
  - relationDeepLevel: Starts at 1 (Root is 0)
*/
func (loadedModel *StatefulModel) mergeRelated(relationDeepLevel byte, documents *wst.A, includeItem wst.IncludeItem, currentContext *EventContext) error {

	if documents == nil {
		return nil
	}

	parentDocs := documents

	relationName := includeItem.Relation
	relation := (*loadedModel.Config.Relations)[relationName]
	relatedModelName := relation.Model
	relatedLoadedModel := (*loadedModel.modelRegistry)[relatedModelName]
	relatedModel := relatedLoadedModel

	parentModel := loadedModel
	parentRelationName := relationName

	if relation.Options.SkipAuth {
		if loadedModel.App.Debug {
			log.Printf("[DEBUG] SkipAuth %v.%v\n", loadedModel.Name, relationName)
		}
	} else {
		// TODO: 01-security/02-authorization/01-permission-propagation.md Defensive programming: verify BaseContext is not nil
		// before dereferencing to prevent potential panic in edge cases
		if currentContext.BaseContext == nil {
			return fmt.Errorf("invalid context: missing base context for permission check on relation %v", relationName)
		}

		objId := "*"
		if len(*documents) == 1 {
			objId = (*documents)[0]["_id"].(primitive.ObjectID).Hex()
		}

		action := fmt.Sprintf("__get__%v", relationName)
		if loadedModel.App.Debug {
			log.Printf("[DEBUG] Check %v.%v\n", loadedModel.Name, action)
		}
		err, allowed := loadedModel.EnforceEx(currentContext.BaseContext.Bearer, objId, action, currentContext.BaseContext)
		if err != nil && err != fiber.ErrUnauthorized {
			return err
		}
		if !allowed || err == fiber.ErrUnauthorized {
			for _, doc := range *documents {
				delete(doc, relationName)
			}
		}
	}

	if relatedModel.Datasource.Name != loadedModel.Datasource.Name {
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

				if !disabledCache && wasEmptyWhere && relatedModel.Config.Cache.Datasource != "" /* && keyFrom == relatedModel.Config.Cache.Keys*/ {

					err := loadedModel.findCachedRelatedDocuments(relatedModel, keyFrom, document, keyTo, targetScope, localCache, cachedRelatedDocs, documentIdx, currentContext)
					if err != nil {
						return err
					}

				}

				relatedInstances := cachedRelatedDocs[documentIdx]
				if relatedInstances == nil {

					(*targetScope.Where)[keyFrom] = document[keyTo]
					if isSingleRelation(relation.Type) {
						targetScope.Limit = 1
					}

					var err error
					relatedInstances, err = relatedLoadedModel.FindMany(targetScope, currentContext).All()
					if err != nil {
						return err
					}
				} else {
					if loadedModel.App.Debug {
						log.Printf("Found cache for %v.%v[%v]\n", loadedModel.Name, relationName, documentIdx)
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
									log.Printf("[WARNING] Invalid type for %v.%v %s\n", loadedModel.Name, relationName, doc[parentRelationName])
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
									log.Println("[WARNING] unknown type for relation", relationName, "in", loadedModel.Name)
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
					err := loadedModel.mergeRelated(relationDeepLevel+1, &nestedDocuments, includeItem, currentContext)
					if err != nil {
						return err
					}

				}
			}
		}
	}

	return nil
}

func (loadedModel *StatefulModel) findCachedRelatedDocuments(relatedLoadedModel *StatefulModel, keyFrom string, document wst.M, keyTo string, targetScope *wst.Filter, localCache map[string]InstanceA, cachedRelatedDocs []InstanceA, documentIdx int, baseContext *EventContext) error {
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
					log.Printf("[DEBUG] cacheLookups %v\n", cacheLookups)
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

				for _, cachedDoc := range cachedDocs {
					cachedInstance, err := relatedLoadedModel.Build(cachedDoc, baseContext)
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
	return nil
}
