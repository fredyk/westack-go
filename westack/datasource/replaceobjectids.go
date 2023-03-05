package datasource

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	wst "github.com/fredyk/westack-go/westack/common"
)

var (
	RegexDaysAgo        = regexp.MustCompile(`^\$(\d+)dago$`)
	RegexWeeksAgo       = regexp.MustCompile(`^\$(\d+)wago$`)
	RegexMonthsAgo      = regexp.MustCompile(`^\$(\d+)mago$`)
	RegexYearsAgo       = regexp.MustCompile(`^\$(\d+)yago$`)
	RegexSecondsAgo     = regexp.MustCompile(`^\$(\d+)Sago$`)
	RegexMinutesAgo     = regexp.MustCompile(`^\$(\d+)Mago$`)
	RegexHoursAgo       = regexp.MustCompile(`^\$(\d+)Hago$`)
	RegexDaysFromNow    = regexp.MustCompile(`^\$(\d+)dfromnow$`)
	RegexWeeksFromNow   = regexp.MustCompile(`^\$(\d+)wfromnow$`)
	RegexMonthsFromNow  = regexp.MustCompile(`^\$(\d+)mfromnow$`)
	RegexYearsFromNow   = regexp.MustCompile(`^\$(\d+)yfromnow$`)
	RegexSecondsFromNow = regexp.MustCompile(`^\$(\d+)Sfromnow$`)
	RegexMinutesFromNow = regexp.MustCompile(`^\$(\d+)Mfromnow$`)
	RegexHoursFromNow   = regexp.MustCompile(`^\$(\d+)Hfromnow$`)
)

func ReplaceObjectIds(data interface{}) interface{} {

	if data == nil {
		return nil
	}

	var finalData wst.M
	switch data.(type) {
	case int, int32, int64, float32, float64, bool, primitive.ObjectID, *primitive.ObjectID, time.Time, primitive.DateTime:
		return data
	case string:
		var newValue interface{}
		var err error
		dataSt := data.(string)
		if wst.RegexpIdEntire.MatchString(dataSt) {
			newValue, err = primitive.ObjectIDFromHex(dataSt)
		} else if wst.IsAnyDate(dataSt) {
			newValue, err = wst.ParseDate(dataSt)
		}
		if err != nil {
			log.Println("WARNING: ", err)
		}
		if newValue != nil {
			return newValue
		} else {
			return data
		}
	case wst.Where:
		finalData = wst.M{}
		for key, value := range data.(wst.Where) {
			finalData[key] = value
		}
		break
	case *wst.Where:
		finalData = wst.M{}
		for key, value := range *data.(*wst.Where) {
			finalData[key] = value
		}
		break
	case map[string]interface{}:
		finalData = wst.M{}
		for key, value := range data.(map[string]interface{}) {
			finalData[key] = value
		}
		break
	case *map[string]interface{}:
		finalData = wst.M{}
		for key, value := range *data.(*map[string]interface{}) {
			finalData[key] = value
		}
		break
	case wst.M:
		finalData = data.(wst.M)
		break
	case *wst.M:
		finalData = *data.(*wst.M)
		break
	default:
		log.Println(fmt.Sprintf("WARNING: Invalid input for ReplaceObjectIds() <- %s", data))
		return data
	}
	for key, value := range finalData {
		if value == nil {
			continue
		}
		var err error
		var newValue interface{}
		if key == "$eq" || key == "$ne" || key == "$gt" || key == "$gte" || key == "$lt" || key == "$lte" {
			if value == "$now" {
				newValue = time.Now()
			} else if value == "$today" {
				newValue = time.Now().Truncate(24 * time.Hour)
			} else if value == "$yesterday" {
				newValue = time.Now().Truncate(24 * time.Hour).Add(-24 * time.Hour)
			} else if value == "$tomorrow" {
				newValue = time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour)
			} else {
				switch value.(type) {
				case string:
					if r := RegexDaysAgo.FindStringSubmatch(value.(string)); len(r) > 1 {
						atoi, err := strconv.Atoi(r[1])
						if err != nil {
							panic(err)
						}
						newValue = time.Now().Add(-time.Duration(atoi) * 24 * time.Hour)
					} else if r := RegexWeeksAgo.FindStringSubmatch(value.(string)); len(r) > 1 {
						atoi, err := strconv.Atoi(r[1])
						if err != nil {
							panic(err)
						}
						newValue = time.Now().Add(-time.Duration(atoi) * 7 * 24 * time.Hour)
					} else if r := RegexMonthsAgo.FindStringSubmatch(value.(string)); len(r) > 1 {
						atoi, err := strconv.Atoi(r[1])
						if err != nil {
							panic(err)
						}
						newValue = time.Now().Add(-time.Duration(atoi) * 30 * 24 * time.Hour)
					} else if r := RegexYearsAgo.FindStringSubmatch(value.(string)); len(r) > 1 {
						atoi, err := strconv.Atoi(r[1])
						if err != nil {
							panic(err)
						}
						newValue = time.Now().Add(-time.Duration(atoi) * 365 * 24 * time.Hour)
					} else if r := RegexSecondsAgo.FindStringSubmatch(value.(string)); len(r) > 1 {
						atoi, err := strconv.Atoi(r[1])
						if err != nil {
							panic(err)
						}
						newValue = time.Now().Add(-time.Duration(atoi) * time.Second)
					} else if r := RegexMinutesAgo.FindStringSubmatch(value.(string)); len(r) > 1 {
						atoi, err := strconv.Atoi(r[1])
						if err != nil {
							panic(err)
						}
						newValue = time.Now().Add(-time.Duration(atoi) * time.Minute)
					} else if r := RegexHoursAgo.FindStringSubmatch(value.(string)); len(r) > 1 {
						atoi, err := strconv.Atoi(r[1])
						if err != nil {
							panic(err)
						}
						newValue = time.Now().Add(-time.Duration(atoi) * time.Hour)
					} else if r := RegexDaysFromNow.FindStringSubmatch(value.(string)); len(r) > 1 {
						atoi, err := strconv.Atoi(r[1])
						if err != nil {
							panic(err)
						}
						newValue = time.Now().Add(time.Duration(atoi) * 24 * time.Hour)
					} else if r := RegexWeeksFromNow.FindStringSubmatch(value.(string)); len(r) > 1 {
						atoi, err := strconv.Atoi(r[1])
						if err != nil {
							panic(err)
						}
						newValue = time.Now().Add(time.Duration(atoi) * 7 * 24 * time.Hour)
					} else if r := RegexMonthsFromNow.FindStringSubmatch(value.(string)); len(r) > 1 {
						atoi, err := strconv.Atoi(r[1])
						if err != nil {
							panic(err)
						}
						newValue = time.Now().Add(time.Duration(atoi) * 30 * 24 * time.Hour)
					} else if r := RegexYearsFromNow.FindStringSubmatch(value.(string)); len(r) > 1 {
						atoi, err := strconv.Atoi(r[1])
						if err != nil {
							panic(err)
						}
						newValue = time.Now().Add(time.Duration(atoi) * 365 * 24 * time.Hour)
					} else if r := RegexSecondsFromNow.FindStringSubmatch(value.(string)); len(r) > 1 {
						atoi, err := strconv.Atoi(r[1])
						if err != nil {
							panic(err)
						}
						newValue = time.Now().Add(time.Duration(atoi) * time.Second)
					} else if r := RegexMinutesFromNow.FindStringSubmatch(value.(string)); len(r) > 1 {
						atoi, err := strconv.Atoi(r[1])
						if err != nil {
							panic(err)
						}
						newValue = time.Now().Add(time.Duration(atoi) * time.Minute)
					} else if r := RegexHoursFromNow.FindStringSubmatch(value.(string)); len(r) > 1 {
						atoi, err := strconv.Atoi(r[1])
						if err != nil {
							panic(err)
						}
						newValue = time.Now().Add(time.Duration(atoi) * time.Hour)
					}
				}
			}
		}
		if newValue == nil {
			switch value.(type) {
			case string, wst.Where, *wst.Where, wst.M, *wst.M, int, int32, int64, float32, float64, bool, primitive.ObjectID, *primitive.ObjectID, time.Time, primitive.DateTime:
				newValue = ReplaceObjectIds(value)
				break
			default:
				asMap, asMapOk := value.(wst.M)
				if asMapOk {
					newValue = ReplaceObjectIds(asMap)
				} else {
					asList, asListOk := value.([]interface{})
					if asListOk {
						for i, asListItem := range asList {
							asList[i] = ReplaceObjectIds(asListItem)
						}
					} else {
						_, asStringListOk := value.([]string)
						if !asStringListOk {
							asMap, asMapOk := value.(map[string]interface{})
							if asMapOk {
								newValue = ReplaceObjectIds(asMap)
							} else {
								asList, asMListOk := value.([]wst.M)
								if asMListOk {
									for i, asListItem := range asList {
										asList[i] = ReplaceObjectIds(asListItem).(wst.M)
									}
								} else {
									log.Println(fmt.Sprintf("WARNING: What to do with %v (%s)?", value, value))
								}
							}
						}
					}
				}
			}
		}
		if err == nil && newValue != nil {
			switch data.(type) {
			case wst.Where:
				data.(wst.Where)[key] = newValue
				break
			case *wst.Where:
				(*data.(*wst.Where))[key] = newValue
				break
			case wst.M:
				data.(wst.M)[key] = newValue
				break
			case *wst.M:
				(*data.(*wst.M))[key] = newValue
				break
			case map[string]interface{}:
				data.(map[string]interface{})[key] = newValue
				break
			case *map[string]interface{}:
				(*data.(*map[string]interface{}))[key] = newValue
				break
			default:
				log.Println(fmt.Sprintf("WARNING: invalid input ReplaceObjectIds() <- %s", data))
				break
			}
		} else if err != nil {
			log.Println("WARNING: ", err)
		}
	}
	return data
}
