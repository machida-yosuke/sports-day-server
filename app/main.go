package main

import (
	"app/mysql"
	"app/typefile"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

func main() {
	db := mysql.SqlConnect()
	// db.AutoMigrate().DropTable(&typefile.Region{}, &typefile.User{}, &typefile.Team{}, &typefile.GameEntry{}, &typefile.GameScore{}, &typefile.Game{})
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
		}

		c.JSON(200, "success")
	})

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

	r.GET("/games", func(c *gin.Context) {
		var games []typefile.Game
		result := db.Find(&games)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"code": "1000", "message": "games Not Found"})
			return
		}

		c.JSON(200, games)
	})

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
