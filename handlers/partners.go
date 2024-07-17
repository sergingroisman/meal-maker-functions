package handlers

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Schedule struct {
	DayOfWeek time.Weekday `bson:"dayOfWeek"`
	StartTime string       `bson:"startTime"`
	EndTime   string       `bson:"endTime"`
}

type Partner struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name      string             `bson:"name" json:"name"`
	CNPJ      string             `bson:"name" json:"cnpj"`
	PartnerID int                `bson:"partnerId" json:"partnerId"`
	Logo      string             `bson:"logo" json:"logo"`
	Schedules []Schedule         `bson:"schedules" json:"schedules"`
	Menus     []Menu             `bson:"menus" json:"menus"`
}

// Verificar se a loja está aberta em um determinado momento
// currentTime := time.Now()
func (partner *Partner) IsOpen(t time.Time) bool {
	for _, schedule := range partner.Schedules {
		startTime := parseTime(schedule.StartTime)
		endTime := parseTime(schedule.EndTime)
		if schedule.DayOfWeek == t.Weekday() {
			if t.After(startTime) && t.Before(endTime) {
				return true
			}
		}
	}
	return false
}

func parseTime(timeStr string) time.Time {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		panic(err)
	}
	return time.Date(0, 0, 0, t.Hour(), t.Minute(), 0, 0, time.Local)
}

func getDayOfWeekString(day time.Weekday) string {
	switch day {
	case time.Monday:
		return "Seg"
	case time.Tuesday:
		return "Ter"
	case time.Wednesday:
		return "Qua"
	case time.Thursday:
		return "Qui"
	case time.Friday:
		return "Sex"
	case time.Saturday:
		return "Sab"
	case time.Sunday:
		return "Dom"
	default:
		return ""
	}
}
