package mysql

import (
	"app/typefile"
	"strconv"
	"strings"

	"github.com/brianvoe/gofakeit/v6"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

type JsonRequest struct {
	RegionId int      `json:"region_id"`
	Users    []string `json:"users"`
}

func Seed(db *gorm.DB) {
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

	for _, region := range regions {
		// 同じ名前が存在する場合はスキップ
		var count int
		db.Model(&typefile.Region{}).Where("name = ?", region).Count(&count)
		if count > 0 {
			continue
		}
		db.Create(&typefile.Region{Name: region})
	}

	for _, game := range games {
		var count int
		db.Model(&typefile.Game{}).Where("name = ?", game).Count(&count)
		if count > 0 {
			continue
		}
		db.Create(&typefile.Game{Name: game})
	}

	for i := 0; i < 2000; i++ {
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
		}
	}
}
