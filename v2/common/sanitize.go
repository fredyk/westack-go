package wst

import (
	"fmt"
	"strings"
)

// AllowedMongoOperators defines the whitelist of safe MongoDB query operators
var AllowedMongoOperators = map[string]bool{
	// Comparison operators
	"$eq":  true,
	"$ne":  true,
	"$gt":  true,
	"$gte": true,
	"$lt":  true,
	"$lte": true,
	"$in":  true,
	"$nin": true,
	
	// Logical operators
	"$and": true,
	"$or":  true,
	"$not": true,
	"$nor": true,
	
	// Element operators
	"$exists": true,
	"$type":   true,
	
	// Array operators
	"$all":        true,
	"$elemMatch":  true,
	"$size":       true,
	
	// Controlled regex (with validation)
	"$regex":    true,
	"$options":  true,
}

// DangerousMongoOperators lists operators that allow code execution
var DangerousMongoOperators = []string{
	"$where",
	"$function",
	"$accumulator",
	"$expr", // Can be dangerous if not controlled
}

// SanitizeMongoQuery validates a MongoDB query object recursively
// Returns error if dangerous operators are found
func SanitizeMongoQuery(query interface{}) error {
	switch v := query.(type) {
	case map[string]interface{}:
		return sanitizeMap(v)
	case M:
		return sanitizeMap(v)
	case Where:
		return sanitizeMap(M(v))
	case *M:
		if v == nil {
			return nil
		}
		return sanitizeMap(*v)
	case *Where:
		if v == nil {
			return nil
		}
		return sanitizeMap(M(*v))
	case []interface{}:
		for _, item := range v {
			if err := SanitizeMongoQuery(item); err != nil {
				return err
			}
		}
		return nil
	case A:
		for _, item := range v {
			if err := SanitizeMongoQuery(item); err != nil {
				return err
			}
		}
		return nil
	default:
		// Primitives are safe
		return nil
	}
}

func sanitizeMap(m map[string]interface{}) error {
	for key, value := range m {
		// Check if key is a MongoDB operator
		if strings.HasPrefix(key, "$") {
			// Check if it's a dangerous operator
			for _, dangerous := range DangerousMongoOperators {
				if key == dangerous {
					return fmt.Errorf("dangerous MongoDB operator not allowed: %s", key)
				}
			}
			
			// Check if it's in the whitelist
			if !AllowedMongoOperators[key] {
				return fmt.Errorf("MongoDB operator not in whitelist: %s", key)
			}
			
			// Special validation for $regex to prevent ReDoS
			if key == "$regex" {
				if regexStr, ok := value.(string); ok {
					if err := validateRegexSafety(regexStr); err != nil {
						return fmt.Errorf("unsafe regex pattern: %w", err)
					}
				}
			}
		}
		
		// Recursively validate nested structures
		if err := SanitizeMongoQuery(value); err != nil {
			return err
		}
	}
	return nil
}

// validateRegexSafety checks for potentially dangerous regex patterns
func validateRegexSafety(pattern string) error {
	// Check for excessive repetition that could cause ReDoS
	dangerousPatterns := []string{
		"(.*)*",     // Nested quantifiers
		"(.+)+",     // Nested quantifiers
		"(.*)+",     // Nested quantifiers
		"(.+)*",     // Nested quantifiers
		"([a-z]*)*", // Nested quantifiers with character class
	}
	
	for _, dangerous := range dangerousPatterns {
		if strings.Contains(pattern, dangerous) {
			return fmt.Errorf("regex pattern contains nested quantifiers (ReDoS risk)")
		}
	}
	
	// Limit pattern length to prevent excessive backtracking
	if len(pattern) > 1000 {
		return fmt.Errorf("regex pattern too long (max 1000 characters)")
	}
	
	return nil
}
