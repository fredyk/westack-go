package model

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

var AuthMutex = sync.RWMutex{}

func (loadedModel *StatefulModel) EnforceEx(token *BearerToken, objId string, action string, eventContext *EventContext) (error, bool) {

	if token != nil && token.Account != nil && token.Account.System == true {
		return nil, true
	}

	if token == nil {
		log.Printf("[WARNING] Trying to enforce without token at %v.%v\n", loadedModel.Name, action)
	}

	var bearerAccountIdSt string
	var targetObjId string

	var locked bool

	if token == nil || token.Account == nil {
		bearerAccountIdSt = "_EVERYONE_"
		targetObjId = "*"
		AuthMutex.RLock()
		if result, isPresent := loadedModel.authCache[bearerAccountIdSt][targetObjId][action]; isPresent {
			if loadedModel.App.Debug || !result {
				log.Printf("[DEBUG] Cache hit for %v.%v ---> %v\n", loadedModel.Name, action, result)
			}
			AuthMutex.RUnlock()
			return nil, result
		} else {
			AuthMutex.RUnlock()
		}

	} else {

		bearerAccountIdSt = token.Raw
		targetObjId = objId

		var created int64
		var ttl int64
		// try to cast
		if v, ok := token.Claims["created"].(int64); ok {
			created = v
		} else {
			created = int64(token.Claims["created"].(float64))
		}
		if v, ok := token.Claims["ttl"].(int64); ok {
			ttl = v
		} else {
			ttl = int64(token.Claims["ttl"].(float64))
		}
		expiresAtTimestamp := created + ttl
		if time.Now().Unix() > expiresAtTimestamp {
			if loadedModel.App.Debug {
				fmt.Println("Token expired for user", bearerAccountIdSt)
			}
			return fiber.ErrUnauthorized, false
		}

		if result, isPresent := loadedModel.authCache[bearerAccountIdSt][targetObjId][action]; isPresent {
			if loadedModel.App.Debug || !result {
				log.Printf("[DEBUG] Cache hit for %v.%v ---> %v\n", loadedModel.Name, action, result)
			}
			return nil, result
		}

		AuthMutex.Lock()
		defer AuthMutex.Unlock()
		locked = true

		_, err := loadedModel.Enforcer.AddRoleForUser(bearerAccountIdSt, "_EVERYONE_")
		if err != nil {
			return err, false
		}
		_, err = loadedModel.Enforcer.AddRoleForUser(bearerAccountIdSt, "_AUTHENTICATED_")
		if err != nil {
			return err, false
		}
		for _, r := range token.Roles {
			_, err := loadedModel.Enforcer.AddRoleForUser(bearerAccountIdSt, r.Name)
			if err != nil {
				return err, false
			}
		}
		err = loadedModel.Enforcer.SavePolicy()
		if err != nil {
			return err, false
		}

	}

	if !locked {
		AuthMutex.Lock()
		defer AuthMutex.Unlock()
	}

	allow, exp, err := loadedModel.Enforcer.EnforceEx(bearerAccountIdSt, targetObjId, action)

	if loadedModel.App.Debug || !allow {
		if len(exp) > 0 {
			log.Println("Explain", exp)
		}
		var subjId string
		if token != nil && token.Account != nil {
			if v, ok := token.Account.Id.(string); ok {
				subjId = v
			} else if vv, ok := token.Account.Id.(primitive.ObjectID); ok {
				subjId = vv.Hex()
			} else {
				subjId = fmt.Sprintf("%v", token.Account.Id)
			}
		}
		fmt.Printf("[DEBUG] EnforceEx for %v.%v (subj=%v,obj=%v) ---> %v\n", loadedModel.Name, action, subjId, targetObjId, allow)
		if eventContext.Remote != nil && eventContext.Remote.Name != "" {
			fmt.Printf("[DEBUG] ... at remote method %v %v%v\n", strings.ToUpper(eventContext.Remote.Http.Verb), loadedModel.BaseUrl, eventContext.Remote.Http.Path)
		}
	}
	if err != nil {
		updateAuthCache(loadedModel, bearerAccountIdSt, targetObjId, action, false)
		return err, false
	}
	if allow {
		updateAuthCache(loadedModel, bearerAccountIdSt, targetObjId, action, true)
		return nil, true
	}
	return fiber.ErrUnauthorized, false
}

func updateAuthCache(loadedModel *StatefulModel, subKey string, targetObjId string, action string, allow bool) {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	if loadedModel.authCache[subKey] == nil {
		addSubjectAuthCacheEntry(loadedModel, subKey)
	}
	if loadedModel.authCache[subKey][targetObjId] == nil {
		addObjectAuthCacheEntry(loadedModel, subKey, targetObjId)
	}
	addActionAuthCacheEntry(loadedModel, subKey, targetObjId, action, allow)
}

var cacheLock = sync.Mutex{}

func addSubjectAuthCacheEntry(loadedModel *StatefulModel, bearerAccountIdSt string) {

	// Delete after 5 minutes
	go func() {
		time.Sleep(5 * time.Minute)
		cacheLock.Lock()
		defer cacheLock.Unlock()
		delete(loadedModel.authCache, bearerAccountIdSt)
	}()

	loadedModel.authCache[bearerAccountIdSt] = make(map[string]map[string]bool)
}

func addObjectAuthCacheEntry(loadedModel *StatefulModel, subKey string, targetObjId string) {
	loadedModel.authCache[subKey][targetObjId] = make(map[string]bool)
}

func addActionAuthCacheEntry(loadedModel *StatefulModel, subKey string, targetObjId string, action string, allow bool) {
	loadedModel.authCache[subKey][targetObjId][action] = allow
}
