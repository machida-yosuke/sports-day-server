package main

import (
	"app/mysql"
	"app/redisClient"
	"app/typefile"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

func calculateCustomScore(score, participants uint, timestamp int64) float64 {
	timeWeight := float64(timestamp) / 1000000000
	return float64(score+participants) + timeWeight
}

func main() {
	var ctx = context.Background()

	db := mysql.SqlConnect()
	rdb := redisClient.RedisConnect()

	// rdb.Set(ctx, "test", "Hello, Redis!", 0)

	// err1 := rdb.Set(ctx, "string", "Hello, Redis v9", 0).Err()
	// if err1 != nil {
	// 	log.Fatalf("Could not set key: %v", err1)
	// }

	// listVal := []string{"val1", "va2", "val3"}
	// err2 := rdb.LPush(ctx, "array", listVal).Err()
	// if err2 != nil {
	// 	log.Fatalf("Could not set key: %v", err2)
	// }

	// db.AutoMigrate().DropTable(&typefile.Region{}, &typefile.User{}, &typefile.Team{}, &typefile.GameEntry{}, &typefile.GameScore{}, &typefile.Game{})
	db.AutoMigrate(&typefile.Region{}, &typefile.User{}, &typefile.Team{}, &typefile.GameEntry{}, &typefile.GameScore{}, &typefile.Game{})
	mysql.Seed(db)
	defer db.Close()

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		// resultString, err1 := rdb.Get(ctx, "string").Result()
		// if err1 != nil {
		// 	c.JSON(400, gin.H{
		// 		"message": "error",
		// 	})
		// }

		// resultArray, err2 := rdb.LRange(ctx, "array", 0, -1).Result()
		// if err2 != nil {
		// 	c.JSON(400, gin.H{
		// 		"message": "error",
		// 	})
		// }

		// resultSet, err3 := rdb.SMembers(ctx, "set").Result()
		// if err3 != nil {
		// 	c.JSON(400, gin.H{
		// 		"message": "error",
		// 	})
		// }

		// resultMysortedset, err4 := rdb.ZRevRangeWithScores(ctx, "mysortedset", 0, -1).Result()
		// if err4 != nil {
		// 	c.JSON(400, gin.H{
		// 		"message": "error",
		// 	})
		// }

		// c.JSON(200, gin.H{
		// 	"resultString":      resultString,
		// 	"resultArray":       resultArray,
		// 	"resultSet":         resultSet,
		// 	"resultMysortedset": resultMysortedset,
		// })

		c.JSON(200, gin.H{
			"message": "Hello, World!",
		})
	})

	r.POST("/teams", func(c *gin.Context) {
		node, err := snowflake.NewNode(1)
		if err != nil {
			panic(err)
		}

		var json typefile.TeamCreateJsonRequest

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
		teamName := strings.Join(json.Users, "/")
		team := typefile.Team{Name: teamName, Region: region, Uuid: uuid.String()}
		db.Create(&team)

		// ユーザーを作成
		for _, userName := range json.Users {
			db.Create(&typefile.User{Name: userName, Team: team})
		}

		c.JSON(200, team)
	})

	r.GET("/teams/:uuid", func(c *gin.Context) {
		var team typefile.Team
		teamResult := db.Preload("Users").Where("uuid = ?", c.Param("uuid")).First(&team)

		if errors.Is(teamResult.Error, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"code": "1000", "message": "team Not Found"})
			return
		}

		c.JSON(200, gin.H{
			"team": team,
		})
	})

	// ゲーム結果の送信
	r.POST("/teams/:uuid/result", func(c *gin.Context) {
		var json typefile.ResultCreateJsonRequest

		var team typefile.Team
		teamResult := db.Preload("Users").Where("uuid = ?", c.Param("uuid")).First(&team)

		if errors.Is(teamResult.Error, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"code": "1000", "message": "team Not Found"})
			return
		}

		if err := c.BindJSON(&json); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		currentTime := time.Now().Unix()

		for _, entry := range json.Entries {
			gameScore := typefile.GameScore{
				Score:       entry.Score,
				HelpScore:   entry.HelpScore,
				HelperCount: entry.HelperCount,
			}

			if err := db.Create(&gameScore).Error; err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			gameEntry := typefile.GameEntry{
				GameID:      entry.GameId,
				TeamID:      team.ID,
				GameScoreID: gameScore.ID,
			}

			if err := db.Create(&gameEntry).Error; err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			customScore := calculateCustomScore(entry.Score, uint(len(team.Users)), currentTime)
			gameName := "game" + fmt.Sprint(entry.GameId)
			err := rdb.ZAdd(ctx, gameName, redis.Z{
				Score:  customScore,
				Member: team.Uuid,
			}).Err()
			if err != nil {
				log.Fatalf("Could not set key: %v", err)
			}
		}

		c.JSON(200, "success")
	})

	// チームの結果取得
	r.GET("/teams/:uuid/result", func(c *gin.Context) {
		var team typefile.Team
		teamResult := db.Preload("Users").Preload("Region").Preload("GameEntries.GameScore").Where("uuid = ?", c.Param("uuid")).First(&team)

		if errors.Is(teamResult.Error, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"code": "1000", "message": "team Not Found"})
			return
		}

		var names []string
		for _, user := range team.Users {
			names = append(names, user.Name)
		}

		var totalScore uint
		for _, entry := range team.GameEntries {
			totalScore += entry.GameScore.Score + entry.GameScore.HelpScore
		}

		var teamResults []struct {
			TeamID     uint
			TotalScore uint
		}

		db.Table("teams").
			Select("teams.id AS team_id, SUM(game_scores.score + game_scores.help_score) as total_score").
			Joins("JOIN game_entries ON teams.id = game_entries.team_id").
			Joins("JOIN game_scores ON game_entries.game_score_id = game_scores.id").
			Group("teams.id").
			Order("total_score DESC").
			Find(&teamResults)

		var teamRank uint
		for i, result := range teamResults {
			if result.TeamID == team.ID {
				teamRank = uint(i + 1)
				break
			}
		}

		gameRanks := []typefile.GameRank{}
		for _, entry := range team.GameEntries {
			var gameRankings []struct {
				TeamID uint
				Score  uint
			}
			db.Table("teams").
				Select("teams.id AS team_id, game_scores.score + game_scores.help_score AS score").
				Joins("JOIN game_entries ON teams.id = game_entries.team_id").
				Joins("JOIN game_scores ON game_entries.game_score_id = game_scores.id").
				Where("game_entries.game_id = ?", entry.GameID).
				Order("score DESC").
				Find(&gameRankings)

			var Rank uint
			for i, result := range gameRankings {
				if result.TeamID == team.ID {
					Rank = uint(i + 1)
					break
				}
			}

			gameRank := typefile.GameRank{
				GameID: entry.GameID,
				Score:  entry.GameScore.Score + entry.GameScore.HelpScore,
				Rank:   Rank,
			}
			gameRanks = append(gameRanks, gameRank)
		}

		var json typefile.ResultJsonResponse
		json.TeamID = team.ID
		json.RegionName = team.Region.Name
		json.UserNames = names
		json.TeamUuid = team.Uuid
		json.TotalScore = totalScore
		json.TeamRank = teamRank
		json.GameRanks = gameRanks

		c.JSON(200, json)
	})

	// ゲーム一覧
	r.GET("/games", func(c *gin.Context) {
		var games []typefile.Game
		result := db.Find(&games)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"code": "1000", "message": "games Not Found"})
			return
		}

		c.JSON(200, games)
	})

	// 所属一覧
	r.GET("/regions", func(c *gin.Context) {
		var regions []typefile.Region
		result := db.Find(&regions)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"code": "1000", "message": "regions Not Found"})
			return
		}

		c.JSON(200, regions)
	})

	r.Run(":8080")
}
