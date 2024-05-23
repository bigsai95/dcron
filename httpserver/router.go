package httpserver

import (
	_ "dcron/docs"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const jsonContentType = "application/json; charset=utf-8"

func InitRouter(ginEngine *gin.Engine) (ginEngineDone *gin.Engine, err error) {

	ginEngine.Use()
	apiEngine := ginEngine.Group("/api")
	apiEngine.GET("/ping", Ping)
	apiEngine.GET("/group/list", ListGroup)
	apiEngine.GET("/game/list", ListGame)
	apiEngine.GET("/job/list", ListJobByGroup)
	apiEngine.GET("/job/match/list", ListJobByMatch)
	apiEngine.GET("/job/game/list", ListJobByGame)
	apiEngine.GET("/job/info", JobInfo)
	apiEngine.GET("/job/query", QueryHandler)
	apiEngine.GET("/job/query/:id", QueryJob)

	apiEngine.POST("/job/add", AddJob)
	apiEngine.POST("/job/replace", ReplaceJob)
	apiEngine.PUT("/job/active/:group/:id", ActiveJob)
	apiEngine.PUT("/job/pause/:group/:id", PauseJob)
	apiEngine.DELETE("/job/delete/:group/:id", DeleteJob)
	apiEngine.DELETE("/jobs/delete/:group", DeleteJobs)
	apiEngine.DELETE("/jobs/delete/:group/:match", DeleteMatchJob)

	apiEngine.POST("/jobs/export", ExportAllHandler)
	apiEngine.POST("/jobs/export/:group", ExportGroupHandler)
	apiEngine.POST("/jobs/export/:group/:match", ExportMatchHandler)
	apiEngine.POST("/jobs/import", ImportHandler)

	apiEngine.POST("/service/cronjob/stop", StopCronJob)
	apiEngine.POST("/service/cronjob/start", StartCronJob)

	ginEngine.GET("/docs/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	ginEngineDone = ginEngine
	return
}
