package main

import (
	"app/mysql"
	"app/typefile"
	"errors"
	"fmt"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

func main() {
	db := mysql.SqlConnect()
	// db.AutoMigrate().DropTable(&Region{}, &User{}, &Team{}, &GameEntry{}, &GameScore{}, &Game{})
	db.AutoMigrate(&typefile.Region{}, &typefile.User{}, &typefile.Team{}, &typefile.GameEntry{}, &typefile.GameScore{}, &typefile.Game{})
	mysql.Seed(db)
	defer db.Close()

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello",
		})
	})

	r.POST("/teams", func(c *gin.Context) {

		node, err := snowflake.NewNode(1)
		if err != nil {
			panic(err)
		}

		var json typefile.JsonRequest

		// リクエストボディをJSONとしてパースして構造体にマッピング
		if err := c.BindJSON(&json); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// 受け取ったデータをログに出力
		fmt.Printf("RegionId: %d, Users: %v\n", json.RegionId, json.Users)

		var region typefile.Region
		if err := db.First(&region, json.RegionId).Error; err != nil {
			c.JSON(400, gin.H{"error": "Region not found"})
			return
		}

		uuid := node.Generate()
		fmt.Println("Generated Snowflake ID:", uuid)

		// チームを作成
		team := typefile.Team{Name: "チーム1", Region: region, Uuid: uuid.String()}
		db.Create(&team)

		// ユーザーを作成
		for _, userName := range json.Users {
			db.Create(&typefile.User{Name: userName, Team: team})
		}

		c.JSON(200, team)
	})

	r.GET("/teams/:uuid", func(c *gin.Context) {
		var team typefile.Team
		result := db.Preload("User, Region").Where("uuid = ?", c.Param("uuid")).First(&team)
		fmt.Println(result.Value)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"code": "1000", "message": "team Not Found"})
			return
		}

		c.JSON(200, result)
	})

	r.GET("/games", func(c *gin.Context) {
		var games []typefile.Game
		// 全て取得
		result := db.Find(&games)
		fmt.Println(result.Value)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"code": "1000", "message": "games Not Found"})
			return
		}

		c.JSON(200, result)
	})

	r.GET("/regions", func(c *gin.Context) {
		var regions []typefile.Region
		result := db.Find(&regions)
		fmt.Println(result.Value)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"code": "1000", "message": "regions Not Found"})
			return
		}

		c.JSON(200, result)
	})

	r.Run(":8080")
}
