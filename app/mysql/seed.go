package mysql

import (
	"app/typefile"

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
}
