package server

import (
	"net/http"

	"ssg/version"

	"github.com/gin-gonic/gin"
)

func Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"version": version.Version(),
	})
}
