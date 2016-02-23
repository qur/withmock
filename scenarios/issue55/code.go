package withdeps

import (
	"github.com/gin-gonic/gin"
)

func TryMe() string {
	return gin.Mode()
}
