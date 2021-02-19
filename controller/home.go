package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-api-frame/tgo"
	"net/http"
	"time"
)

func Welcome(c *gin.Context) {
	now := time.Now().String()
	sysName := tgo.ConfigAppGetString("sysname", "default service")
	content := fmt.Sprintf("Welcome to %s@%s", sysName, now)
	c.String(http.StatusOK, content)
}
