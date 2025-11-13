# Gin Default Server

This is API experiment for Gin.

```go
package main

import (
	"EasyTools/app/controller/connect/ssh/gin"
	"EasyTools/app/connect/ssh/gin/ginS"
)

func main() {
	ginS.GET("/", func(c *gin.Context) { c.String(200, "Hello World") })
	ginS.Run()
}
```
