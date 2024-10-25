package seed

import (
	"app/typefile"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/redis/go-redis/v9"
)

func calculateCustomScore(score, participants uint, timestamp int64) float64 {
	timeWeight := float64(timestamp) / 1000000000
	return float64(score+participants) + timeWeight
}

type JsonRequest struct {
	RegionId int      `json:"region_id"`
	Users    []string `json:"users"`
}

func Seed(db *gorm.DB, rdb *redis.Client) {
	var ctx = context.Background()
	regions := []string{
		"所属A",
		"所属B",
		"所属C",
		"所属D",
		"所属E",
	}

	games := []string{
		"ゲームA",
		"ゲームB",
		"ゲームC",
		"ゲームD",
	}

	// リージョンの登録
	for _, region := range regions {
		// 同じ名前が存在する場合はスキップ
		var count int
		db.Model(&typefile.Region{}).Where("name = ?", region).Count(&count)
		if count > 0 {
			continue
		}
		db.Create(&typefile.Region{Name: region})
	}

	// ゲームの登録
	for _, game := range games {
		var count int
		db.Model(&typefile.Game{}).Where("name = ?", game).Count(&count)
		if count > 0 {
			continue
		}
		db.Create(&typefile.Game{Name: game})
	}

	// 適当なチームを作成
	for i := 0; i < 10; i++ {
		// 同じuuidが存在する場合はスキップ
		var count int
		db.Model(&typefile.Team{}).Where("uuid = ?", strconv.Itoa(i)).Count(&count)
		if count > 0 {
			continue
		}

		// リージョンをランダムで取得
		var dbRegions []typefile.Region
		db.Find(&dbRegions)
		region := dbRegions[i%len(dbRegions)]

		var userNames = []string{gofakeit.Name(), gofakeit.Name(), gofakeit.Name()}
		team := typefile.Team{Name: strings.Join(userNames, "/"), Region: region, Uuid: strconv.Itoa(i)}
		db.Create(&team)

		// ユーザーを作成
		for _, userName := range userNames {
			db.Create(&typefile.User{Name: userName, Team: team})
		}

		var dbGames []typefile.Game
		db.Find(&dbGames)

		currentTime := gofakeit.DateRange(time.Now(), time.Now().AddDate(0, 0, -1))

		for _, game := range dbGames {
			gameScore := typefile.GameScore{
				Score:       uint(gofakeit.IntRange(0, 3000)),
				HelpScore:   uint(gofakeit.IntRange(0, 200)),
				HelperCount: uint(gofakeit.IntRange(0, 1)),
			}
			db.Create(&gameScore)

			gameEntry := typefile.GameEntry{
				GameID:      game.ID,
				TeamID:      team.ID,
				GameScoreID: gameScore.ID,
			}

			db.Create(&gameEntry)

			customScore := calculateCustomScore(gameScore.Score+gameScore.HelpScore, uint(len(team.Users)), currentTime.Unix())
			gameName := "game" + fmt.Sprint(game.ID)
			err := rdb.ZAdd(ctx, gameName, redis.Z{
				Score:  customScore,
				Member: team.Uuid,
			}).Err()
			if err != nil {
				fmt.Println("redis error" + err.Error())
			}
		}

		gameEntries := []typefile.GameEntry{}
		db.Preload("GameScore").Where("team_id = ?", team.ID).Find(&gameEntries)

		totalScore := uint(0)
		totalHelperCount := uint(0)
		for _, entry := range gameEntries {
			totalScore += entry.GameScore.Score + entry.GameScore.HelpScore
			totalHelperCount += entry.GameScore.HelperCount
		}

		customScore := calculateCustomScore(totalScore, totalHelperCount, currentTime.Unix())
		err := rdb.ZAdd(ctx, "total", redis.Z{
			Score:  customScore,
			Member: team.Uuid,
		}).Err()
		if err != nil {
			fmt.Println("redis error" + err.Error())
		}
	}
}
