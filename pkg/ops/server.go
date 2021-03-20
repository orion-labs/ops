package ops

import (
	"embed"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/fs"
	"log"
	"net/http"
)

//go:embed views
var content embed.FS

func RunServer(address string, port int) (err error) {
	router := gin.Default()

	api := router.Group("/api")

	{
		api.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "pong",
			})
		})
	}

	api.GET("/systems", SystemHandler)
	//api.POST("/jokes/like/:jokeID", LikeJoke)

	router.Use(Serve("/", content))

	addr := fmt.Sprintf("%s:%d", address, port)
	fmt.Printf("Server starting on %s.\n", addr)

	err = router.Run(addr)

	return err
}

func Serve(urlPrefix string, efs embed.FS) gin.HandlerFunc {
	// the embedded filesystem has a 'views/' at the top level.  We wanna strip this so we can treat the root of the views directory as the web root.
	fsys, err := fs.Sub(efs, "views")
	if err != nil {
		log.Fatalf(err.Error())
	}

	fileserver := http.FileServer(http.FS(fsys))
	if urlPrefix != "" {
		fileserver = http.StripPrefix(urlPrefix, fileserver)
	}

	return func(c *gin.Context) {
		fileserver.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

func SystemHandler(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, stacks)
}

//func LikeJoke(c *gin.Context) {
//	if jokeid, err := strconv.Atoi(c.Param("jokeID")); err == nil {
//		for i:=0; i < len(jokes); i++ {
//			if jokes[i].ID == jokeid {
//				jokes[i].Likes += 1
//			}
//		}
//
//		c.JSON(http.StatusOK, &jokes)
//	} else {
//		c.AbortWithStatus(http.StatusNotFound)
//	}
//}

// stack name , account, statuses,

var stacks = []Stack{}
