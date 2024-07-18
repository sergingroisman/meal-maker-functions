package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Schedule struct {
	DayOfWeek string `bson:"day_of_week" json:"day_of_week"`
	StartTime string `bson:"start_time" json:"start_time"`
	EndTime   string `bson:"end_time" json:"end_time"`
}

type Partner struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name      string             `bson:"name" json:"name"`
	CNPJ      string             `bson:"cnpj" json:"cnpj"`
	PartnerID int                `bson:"partner_id" json:"partner_id"`
	Logo      string             `bson:"logo" json:"logo"`
	Schedules []Schedule         `bson:"schedules" json:"schedules"`
	Menus     []Menu             `bson:"menus" json:"menus"`
	CreatedAt string             `bson:"created_at" json:"created_at"`
	UpdatedAt string             `bson:"updated_at" json:"updated_at"`
}

type PartnerBFFResponse struct {
	Name      string     `json:"name"`
	CNPJ      string     `json:"cnpj"`
	PartnerID int        `json:"partner_id"`
	Logo      string     `json:"logo"`
	IsOpen    bool       `json:"is_open"`
	Schedules []Schedule `json:"schedules"`
	Menus     []Menu     `json:"menus"`
}

func (h *Handlers) GetRestaurantByPartnerId(c *gin.Context) {
	partner_id_str := c.Param("partner_id")
	if partner_id_str == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar o id do parceciro como parâmetro")
		return
	}

	partner_id, err := strconv.Atoi(partner_id_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Formato de id inválido",
		})
		return
	}

	var partner Partner
	collection := h.database.Collection("Partners")
	filter := bson.D{{Key: "partner_id", Value: partner_id}}
	err = collection.FindOne(h.context, filter).Decode(&partner)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusNotFound, gin.H{
			"status_code": http.StatusNotFound,
			"message":     "Não foi possível encontrar um parceiro com esse id",
		})
		return
	}

	current_time := time.Now()
	restaurant := PartnerBFFResponse{
		Name:      partner.Name,
		CNPJ:      partner.CNPJ,
		PartnerID: partner.PartnerID,
		Logo:      partner.Logo,
		IsOpen:    partner.IsOpen(current_time),
		Schedules: partner.Schedules,
		Menus:     partner.Menus,
	}

	c.IndentedJSON(http.StatusOK, restaurant)
}

func (partner *Partner) IsOpen(t time.Time) bool {
	for _, schedule := range partner.Schedules {
		startTime := parseTime(schedule.StartTime)
		endTime := parseTime(schedule.EndTime)
		if parseDayOfWeekTime(schedule.DayOfWeek) == t.Weekday() {
			if t.After(startTime) && t.Before(endTime) {
				return true
			}
		}
	}
	return false
}

func parseTime(time_str string) time.Time {
	t, err := time.Parse("15:04", time_str)
	if err != nil {
		panic(err)
	}
	return time.Date(0, 0, 0, t.Hour(), t.Minute(), 0, 0, time.Local)
}

func parseDayOfWeekTime(day string) time.Weekday {
	dayToLowerCase := strings.ToLower(day)
	switch dayToLowerCase {
	case "monday":
		return time.Monday
	case "tuesday":
		return time.Tuesday
	case "wednesday":
		return time.Wednesday
	case "thursday":
		return time.Thursday
	case "friday":
		return time.Friday
	case "saturday":
		return time.Saturday
	case "sunday":
		return time.Sunday
	default:
		return time.Monday
	}
}

func parseDayOfWeekString(day time.Weekday) string {
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
