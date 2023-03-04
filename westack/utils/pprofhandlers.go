package utils

import (
	"encoding/base64"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/pprof"
)

type BasicAuthOptions struct {
	Username string
	Password string
}

type PprofMiddleOptions struct {
	Auth BasicAuthOptions
}

func PprofHandlers(options PprofMiddleOptions) func(ctx *fiber.Ctx) error {
	pprofHandler := pprof.New()
	return func(ctx *fiber.Ctx) error {
		// check if url starts with /debug/pprof
		if regexp.MustCompile("^/debug/pprof").MatchString(ctx.OriginalURL()) {
			// check basic auth
			auth := ctx.Get("Authorization")

			if auth == "" {
				ctx.Status(fiber.StatusUnauthorized)
				ctx.Set("WWW-Authenticate", "Basic realm=\"Restricted\"")
				return ctx.Send([]byte("Unauthorized"))
			}

			// parse the basic auth
			user, pass, ok := parseBasicAuth(auth)
			if !ok {
				ctx.Status(fiber.StatusUnauthorized)
				ctx.Set("WWW-Authenticate", "Basic realm=\"Restricted\"")
				return ctx.Send([]byte("Unauthorized"))
			}

			// check if the user and password are correct
			if user != options.Auth.Username || pass != options.Auth.Password {
				ctx.Status(fiber.StatusUnauthorized)
				ctx.Set("WWW-Authenticate", "Basic realm=\"Restricted\"")
				return ctx.Send([]byte("Unauthorized"))
			}

			return pprofHandler(ctx)
		}
		return ctx.Next()
	}
}

func parseBasicAuth(auth string) (user string, pass string, ok bool) {
	if !strings.HasPrefix(auth, "Basic ") {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}
