package server

import (
	"net/http"

	"ssg/version"

	"github.com/gin-gonic/gin"
)

func Status(c *gin.Context) {
	debug := false
	if raw, ok := c.Get("server"); ok {
		if s, ok := raw.(*Server); ok {
			debug = s.DebugEnabled()
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"version": version.Version(),
		"debug":   debug,
	})
}
