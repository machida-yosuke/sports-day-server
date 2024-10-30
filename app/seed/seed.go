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
	regions := []typefile.Region{
		{
			Name:     "所属A",
			Hiragana: "しょぞくA",
			Katakana: "ショゾクA",
			Alphabet: "shozokuA",
			Place:    "placeA",
		},
		{
			Name:     "所属B",
			Hiragana: "しょぞくB",
			Katakana: "ショゾクB",
			Alphabet: "shozokuB",
			Place:    "placeB",
		},
		{
			Name:     "所属C",
			Hiragana: "しょぞくC",
			Katakana: "ショゾクC",
			Alphabet: "shozokuC",
			Place:    "placeC",
		},
		{
			Name:     "所属D",
			Hiragana: "しょぞくD",
			Katakana: "ショゾクD",
			Alphabet: "shozokuD",
			Place:    "placeD",
		},
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
		db.Create(&typefile.Region{Name: region.Name, Hiragana: region.Hiragana, Katakana: region.Katakana, Alphabet: region.Alphabet, Place: region.Place})
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

		// チームランキング用のスコアを登録
		customScore := calculateCustomScore(totalScore, totalHelperCount, currentTime.Unix())
		err := rdb.ZAdd(ctx, "total", redis.Z{
			Score:  customScore,
			Member: team.Uuid,
		}).Err()
		if err != nil {
			fmt.Println("redis error" + err.Error())
		}

		// リージョンのチームの上位１０チームのスコア用のスコアを登録
		err = rdb.ZAdd(ctx, "region-"+fmt.Sprint(region.ID)+"-teams-total-score", redis.Z{
			Score:  float64(totalScore),
			Member: team.Uuid,
		}).Err()
		if err != nil {
			fmt.Println("redis error" + err.Error())
		}

		// リージョンのチーム上位10チームのスコアを取得
		regionTeamTotalScore := uint(0)
		teamsScore, errTotal := rdb.ZRevRangeWithScores(ctx, "region-"+fmt.Sprint(team.RegionID)+"-teams-total-score", 0, 10).Result()
		if errTotal != nil {
			fmt.Println("redis error" + err.Error())
		}
		for _, teamScore := range teamsScore {
			regionTeamTotalScore += uint(teamScore.Score)
		}

		// リージョンのチーム全員のスコアを登録
		err = rdb.ZAdd(ctx, "region-top-10-teams-total-score", redis.Z{
			Score:  float64(regionTeamTotalScore),
			Member: team.RegionID,
		}).Err()
		if err != nil {
			fmt.Println("redis error" + err.Error())
		}
	}
}
