package typefile

import "github.com/jinzhu/gorm"

type Region struct {
	gorm.Model
	Name     string
	Hiragana string
	Katakana string
	Alphabet string
	Place    string
	Teams    []Team `gorm:"foreignKey:RegionID"`
}

type Team struct {
	gorm.Model
	Name        string
	Uuid        string
	RegionID    uint
	Region      Region
	Users       []User
	GameEntries []GameEntry
}

type User struct {
	gorm.Model
	Name   string
	TeamID uint
	Team   Team
}

type GameEntry struct {
	gorm.Model
	TeamID      uint `gorm:"index"`
	Team        Team
	GameID      uint `gorm:"index"`
	Game        Game
	GameScoreID uint
	GameScore   GameScore
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
	RegionID   uint   `json:"region_id"`
	TeamUuid   string `json:"team_uuid"`
	RegionName string `json:"region_name"`

	UserNames                       []string   `json:"user_names"`
	TotalScore                      uint       `json:"total_score"`
	TeamRank                        uint       `json:"team_rank"`
	ALLTeamCount                    uint       `json:"all_team_count"`
	GameRanks                       []GameRank `json:"game_ranks"`
	Top10TeamsScoreRankByRegion     uint       `json:"top10_teams_score_rank_by_region"`
	EnteredRegionCountInCompetition uint       `json:"entered_region_count_in_competition"`
}

type GameRank struct {
	GameID uint `json:"game_id"`
	Score  uint `json:"score"`
	Rank   uint `json:"rank"`
}

type Pagination struct {
	Offset int
	Limit  int
}
