package typefile

import "github.com/jinzhu/gorm"

type Region struct {
	gorm.Model
	Name  string
	Teams []Team `gorm:"foreignKey:RegionID"`
}

type User struct {
	gorm.Model
	Name   string
	TeamID uint
	Team   Team
}

type Team struct {
	gorm.Model
	Name        string
	Uuid        string
	RegionID    uint
	Region      Region
	Users       []User      `gorm:"foreignKey:TeamID"`
	GameEntries []GameEntry `gorm:"foreignKey:TeamID"`
}

type GameEntry struct {
	gorm.Model
	TeamID      uint
	GameID      uint
	Game        Game
	Team        Team
	GameScoreID uint
	GameScore   GameScore
}

type GameScore struct {
	gorm.Model
	Score uint
}

type Game struct {
	gorm.Model
	Name string
}

type JsonRequest struct {
	RegionId int      `json:"region_id"`
	Users    []string `json:"users"`
}
