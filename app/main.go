package main

import (
	"app/mysql"
	"app/redisClient"
	"app/seed"
	"app/typefile"
	"context"
	"errors"
	"fmt"
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

	// db.AutoMigrate().DropTable(&typefile.Region{}, &typefile.User{}, &typefile.Team{}, &typefile.GameEntry{}, &typefile.GameScore{}, &typefile.Game{})
	db.AutoMigrate(&typefile.Region{}, &typefile.User{}, &typefile.Team{}, &typefile.GameEntry{}, &typefile.GameScore{}, &typefile.Game{})
	seed.Seed(db, rdb)
	defer db.Close()

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		resultGame1, _ := rdb.ZRevRangeWithScores(ctx, "game1", 0, -1).Result()
		resultGame2, _ := rdb.ZRevRangeWithScores(ctx, "game2", 0, -1).Result()
		resultGame3, _ := rdb.ZRevRangeWithScores(ctx, "game3", 0, -1).Result()
		resultGame4, _ := rdb.ZRevRangeWithScores(ctx, "game4", 0, -1).Result()
		resultTotal, _ := rdb.ZRevRangeWithScores(ctx, "total", 0, -1).Result()
		regionTop10TeamsTotalScore, _ := rdb.ZRevRangeWithScores(ctx, "region-top-10-teams-total-score", 0, -1).Result()

		c.JSON(200, gin.H{
			"regionTop10TeamsTotalScore": regionTop10TeamsTotalScore,
			"resultGame1":                resultGame1,
			"resultGame2":                resultGame2,
			"resultGame3":                resultGame3,
			"resultGame4":                resultGame4,
			"resultTotal":                resultTotal,
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
				c.JSON(500, gin.H{"error": "redis error" + err.Error()})
				return
			}
		}

		totalScore := uint(0)
		totalHelperCount := uint(0)
		for _, entry := range json.Entries {
			totalScore += entry.Score + entry.HelpScore
			totalHelperCount += entry.HelperCount
		}

		customScore := calculateCustomScore(totalScore, totalHelperCount, currentTime)
		err := rdb.ZAdd(ctx, "total", redis.Z{
			Score:  customScore,
			Member: team.Uuid,
		}).Err()
		if err != nil {
			c.JSON(500, gin.H{"error": "redis error" + err.Error()})
			return
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

		// 全チームのランキングを取得
		resultTotal, errTotal := rdb.ZRevRangeWithScores(ctx, "total", 0, -1).Result()
		if errTotal != nil {
			c.JSON(400, gin.H{
				"message": "error",
			})
		}

		teamRank := uint(0)
		for i, result := range resultTotal {
			if result.Member == team.Uuid {
				teamRank = uint(i + 1)
				break
			}
		}

		// ゲームごとのランキングを取得
		gameRanks := []typefile.GameRank{}
		for _, entry := range team.GameEntries {
			key := "game" + fmt.Sprint(entry.GameID)
			resultGame, err := rdb.ZRevRangeWithScores(ctx, key, 0, -1).Result()
			if err != nil {
				c.JSON(400, gin.H{
					"message": "error",
				})
			}
			var Rank uint
			for i, result := range resultGame {
				if result.Member == team.Uuid {
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

		regionsTotalScore, errRegionsTotalScore := rdb.ZRevRangeWithScores(ctx, "region-top-10-teams-total-score", 0, -1).Result()
		if errRegionsTotalScore != nil {
			c.JSON(400, gin.H{
				"message": "error",
			})
		}
		enteredRegionCountInCompetition := uint(len(regionsTotalScore))

		top10TeamsScoreRankByRegion := uint(0)
		for i, result := range regionsTotalScore {
			if result.Member == fmt.Sprint(team.RegionID) {
				top10TeamsScoreRankByRegion = uint(i + 1)
				break
			}
		}

		var json typefile.ResultJsonResponse
		json.TeamID = team.ID
		json.RegionID = team.RegionID
		json.RegionName = team.Region.Name
		json.UserNames = names
		json.TeamUuid = team.Uuid
		json.TotalScore = totalScore
		json.TeamRank = teamRank
		json.ALLTeamCount = uint(len(resultTotal))
		json.GameRanks = gameRanks
		json.Top10TeamsScoreRankByRegion = top10TeamsScoreRankByRegion
		json.EnteredRegionCountInCompetition = enteredRegionCountInCompetition
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
