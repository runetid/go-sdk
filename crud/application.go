package crud

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/runetid/go-sdk"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http"
	"os"
	"sync/atomic"
)

type Application struct {
	Router *gin.Engine
	Db     *gorm.DB
}

func (a Application) Run() {
	isReady := &atomic.Value{}
	isReady.Store(false)
	sdk.AppendMetrics(a.Router)

	a.Router.GET("/healthz", sdk.HealthzWithDb(a.Db))
	a.Router.GET("/readyz", gin.WrapF(sdk.Readyz(isReady)))

	a.Router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Page not found"})
	})

	done := make(chan bool)
	go a.Router.Run(os.Getenv("HTTP_ADDR"))
	isReady.Store(true)
	<-done
}

type CrudModel interface {
	List(db *gorm.DB, request ListRequest, params ...FilterParams) (interface{}, int64, error)
	GetFilterParams(c *gin.Context) []FilterParams
	Create(db *gorm.DB) (interface{}, error)
	Update(db *gorm.DB) (interface{}, error)
	DecodeCreate(c *gin.Context) (interface{}, error)
	Delete(db *gorm.DB, key string) bool
	Get(db *gorm.DB, key string) (interface{}, error)
}

type BaseCrudModel struct {
}

func (u BaseCrudModel) GetFilterParams(c *gin.Context) []FilterParams {
	var p []FilterParams
	return p
}

func (u BaseCrudModel) DecodeCreate(c *gin.Context) interface{} {
	return c.Bind(u)
}

func (a Application) AppendListEndpoint(prefix string, entity CrudModel) {
	a.Router.GET(prefix+"/list", func(c *gin.Context) {

		var request ListRequest
		err := c.ShouldBindQuery(&request)
		if err != nil {
			c.JSON(500, gin.H{"message": "Wrong limit or offset params " + err.Error()})
			c.Writer.WriteHeaderNow()
			c.Abort()
			return
		}

		var m interface{}
		var cnt int64

		m, cnt, err = entity.List(a.Db, request, entity.GetFilterParams(c)...)

		c.JSON(200, gin.H{"data": m, "error": err, "total": cnt})
		return
	})
}

func (a Application) AppendCreateEndpoint(prefix string, entity CrudModel) {
	a.Router.POST(prefix+"/", func(c *gin.Context) {
		decode, _ := entity.DecodeCreate(c)
		m, err := decode.(CrudModel).Create(a.Db)

		c.JSON(200, gin.H{"data": m, "error": err})
		return
	})
}

func (a Application) AppendUpdateEndpoint(prefix string, entity CrudModel) {
	a.Router.PATCH(prefix+"/", func(c *gin.Context) {
		decode, _ := entity.DecodeCreate(c)
		m, err := decode.(CrudModel).Update(a.Db)

		c.JSON(200, gin.H{"data": m, "error": err})
		return
	})
}

func (a Application) AppendDeleteEndpoint(prefix string, entity CrudModel) {
	a.Router.DELETE(prefix+"/", func(c *gin.Context) {

		if entity.Delete(a.Db, c.Param("id")) {

			c.JSON(http.StatusOK, gin.H{"message": "ok"})
			return
		}

		c.JSON(http.StatusConflict, gin.H{"message": "cant delete"})
		return
	})
}

func (a Application) AppendGetEndpoint(prefix string, entity CrudModel) {
	a.Router.GET(prefix+"/", func(c *gin.Context) {
		model, err := entity.Get(a.Db, c.Param("id"))

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"data": model, "error": err})
			return
		}

		c.JSON(200, gin.H{"data": model, "error": err})
		return
	})
}

func NewCrudApplication() (*Application, error) {

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Europe/Moscow",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	r := gin.Default()
	r.Use(sdk.CorsMiddleware())
	r.Use(sdk.JsonMiddleware())
	r.Use(sdk.DbMiddleware(db))
	//r.Use(sdk.ApiMiddleware(db))

	return &Application{
		Router: r,
		Db:     db,
	}, err
}

type ListRequest struct {
	Limit  int `form:"limit" binding:"required,number,min=1,max=100"`
	Offset int `form:"offset" binding:"number"`
}

type FilterParams struct {
	Key      string
	Value    string
	Operator string
}