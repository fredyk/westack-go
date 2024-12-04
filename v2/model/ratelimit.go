package model

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strings"
	"sync"
	"time"
)

var WhiteListedUsers = map[string]bool{}
var whileListedUsersMutex = &sync.RWMutex{}

type RateLimit struct {
	Name string
	// The number of requests allowed in the time period
	MaxRequests int
	// The time period in which the requests are allowed
	TimePeriod time.Duration
	// The number of requests made in the current time period
	RequestsMadeByIp map[string]int
	// The time at which the current time period started
	StartTime time.Time

	bearerCache           map[string]bool
	requestsMadeByIpMutex *sync.RWMutex
	bearerCacheMutex      *sync.RWMutex
	whileListAdmins       bool
}

func (rateLimit *RateLimit) Allow(eventContext *EventContext) bool {

	var fiberContext *fiber.Ctx
	var bearer *BearerToken
	var isAdmin bool
	var baseContext = eventContext

	for baseContext.BaseContext != nil {
		baseContext = baseContext.BaseContext
	}
	if baseContext.Ctx != nil {
		fiberContext = baseContext.Ctx
	}
	if baseContext.Bearer != nil {
		bearer = baseContext.Bearer
		if bearer != nil && bearer.Account != nil {
			userIdAsSt := ""
			if bearer.Account.Id != nil {
				if id, ok := bearer.Account.Id.(string); ok {
					userIdAsSt = id
				} else if id, ok := bearer.Account.Id.(primitive.ObjectID); ok {
					userIdAsSt = id.Hex()
				}
				if isWhiteListed(userIdAsSt, rateLimit) {
					return true
				}
			}
		}
		isAdmin = rateLimit.IsAdmin(bearer)
	}

	if isAdmin {
		fmt.Printf("[DEBUG] [%s] Admin request allowed\n", rateLimit.Name)
		return true
	}

	var ips = ""
	if fiberContext != nil {
		if len(fiberContext.IPs()) > 0 {
			ips = strings.Join(fiberContext.IPs(), ",")
		} else {
			ips = fiberContext.IP()
		}
	}

	//// vpn traffic allowed
	//if ips == "<my_vpn_ip>" || ips == "<my_vpn_ip>,<other_vpn_ip>" {
	//	fmt.Printf("[DEBUG] [%s] VPN traffic allowed\n", rateLimit.Name)
	//	return true
	//} else {
	//	//fmt.Printf("[DEBUG] VPN traffic not allowed for %s\n", ips)
	//}

	rateLimit.requestsMadeByIpMutex.Lock()
	defer rateLimit.requestsMadeByIpMutex.Unlock()

	if _, ok := rateLimit.RequestsMadeByIp[ips]; !ok {
		rateLimit.RequestsMadeByIp[ips] = 0
	}

	if time.Since(rateLimit.StartTime) > rateLimit.TimePeriod {
		rateLimit.StartTime = time.Now()
		rateLimit.RequestsMadeByIp[ips] = 0
	}

	if rateLimit.RequestsMadeByIp[ips] >= rateLimit.MaxRequests {
		fmt.Printf("[DEBUG] [%s] Too many requests by %s: %d\n", rateLimit.Name, ips, rateLimit.RequestsMadeByIp[ips])
		return false
	}

	rateLimit.RequestsMadeByIp[ips]++

	return true
}

func isWhiteListed(userId string, rateLimit *RateLimit) bool {
	if userId == "" {
		return false
	}
	isWhiteListed := false
	whileListedUsersMutex.RLock()
	if WhiteListedUsers[userId] {
		whileListedUsersMutex.RUnlock()
		fmt.Printf("[%s] White listed user allowed\n", rateLimit.Name)
		isWhiteListed = true
	}
	whileListedUsersMutex.RUnlock()
	return isWhiteListed
}

func (rateLimit *RateLimit) IsAdmin(bearerToken *BearerToken) bool {
	if !rateLimit.whileListAdmins {
		return false
	}
	if bearerToken == nil {
		return false
	}
	rateLimit.bearerCacheMutex.RLock()
	if isAdmin, ok := rateLimit.bearerCache[bearerToken.Raw]; ok {
		rateLimit.bearerCacheMutex.RUnlock()
		return isAdmin
	}
	rateLimit.bearerCacheMutex.RUnlock()
	if bearerToken.Account == nil {
		return false
	}
	if bearerToken.Account.System {
		rateLimit.bearerCacheMutex.Lock()
		rateLimit.bearerCache[bearerToken.Raw] = true
		rateLimit.bearerCacheMutex.Unlock()
		return true
	}
	// iterate roles
	for _, role := range bearerToken.Roles {
		if role.Name == "admin" {
			rateLimit.bearerCacheMutex.Lock()
			rateLimit.bearerCache[bearerToken.Raw] = true
			rateLimit.bearerCacheMutex.Unlock()
			return true
		}
	}
	rateLimit.bearerCacheMutex.Lock()
	rateLimit.bearerCache[bearerToken.Raw] = false
	rateLimit.bearerCacheMutex.Unlock()
	return false
}

func NewRateLimit(name string, maxRequests int, timePeriod time.Duration, whiteListAdmins bool) *RateLimit {
	return &RateLimit{
		Name:                  name,
		MaxRequests:           maxRequests,
		TimePeriod:            timePeriod,
		RequestsMadeByIp:      map[string]int{},
		StartTime:             time.Now(),
		requestsMadeByIpMutex: &sync.RWMutex{},
		bearerCache:           map[string]bool{},
		bearerCacheMutex:      &sync.RWMutex{},
		whileListAdmins:       whiteListAdmins,
	}
}

func WhiteList(userId string) {
	whileListedUsersMutex.Lock()
	WhiteListedUsers[userId] = true
	whileListedUsersMutex.Unlock()
}
