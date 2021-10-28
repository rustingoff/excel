package main

import (
	"github.com/gin-gonic/gin"
	server "github.com/rustingoff/excel"
	"github.com/rustingoff/excel/database"
	"github.com/rustingoff/excel/handler"
)

var (
	elasticClient = database.NewElasticSearchConnection()

	handlers = handler.GetHandlerPackage(elasticClient)
)

func main() {
	router := gin.Default()
	router.Use(gin.Recovery())

	router.LoadHTMLFiles("template/index.html", "template/post.html", "template/show.html")
	router.Static("/static/", "static/exports")
	router.GET("/", handlers.Home)

	router.GET("/campaign", handlers.Campaign)
	router.POST("/campaign", handlers.Campaign)
	router.GET("/show/campaigns", handlers.Show)
	router.POST("/export/:id", handlers.Export)
	router.GET("/delete/:id", handlers.Delete)

	srv := new(server.Server)
	if err := srv.Run(":8080", router); err != nil {
		panic(err)
	}
}
