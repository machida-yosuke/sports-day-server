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
	TeamID      uint `gorm:"index"`
	Team        Team `gorm:"foreignKey:TeamID"`
	GameID      uint `gorm:"index"`
	Game        Game `gorm:"foreignKey:GameID"`
	GameScoreID uint
	GameScore   GameScore `gorm:"foreignKey:GameScoreID"`
}

type GameScore struct {
	gorm.Model
	Score       uint
	HelpScore   uint
	HelperCount uint
}

type Game struct {
	gorm.Model
	Name string
}

type TeamCreateJsonRequest struct {
	RegionId int      `json:"region_id"`
	Users    []string `json:"users"`
}

type ResultCreateJsonRequest struct {
	Entries []GameEntryJson `json:"entries"`
}

type GameEntryJson struct {
	GameId      uint `json:"game_id"`
	Score       uint `json:"score"`
	HelpScore   uint `json:"help_score"`
	HelperCount uint `json:"helper_count"`
}

type ResultJsonResponse struct {
	TeamID     uint   `json:"team_id"`
	TeamUuid   string `json:"team_uuid"`
	RegionName string `json:"region_name"`

	UserNames  []string   `json:"user_names"`
	TotalScore uint       `json:"total_score"`
	TeamRank   uint       `json:"team_rank"`
	GameRanks  []GameRank `json:"game_ranks"`

	RegionTotalUserCount            uint `json:"region_total_user_count"`
	Top10TeamsScoreRankingByRegion  uint `json:"top10_teams_score_ranking_by_region"`
	EnteredRegionCountInCompetition uint `json:"entered_region_count_in_competition"`
}

type GameRank struct {
	GameID uint `json:"game_id"`
	Score  uint `json:"score"`
	Rank   uint `json:"rank"`
}
