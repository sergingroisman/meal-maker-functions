package handlers

import (
	"log"
	"net/http"
	"strconv"
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
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name        string             `bson:"name" json:"name"`
	PartnerID   int                `bson:"partner_id" json:"partner_id"`
	Logo        string             `bson:"logo" json:"logo"`
	Schedules   []Schedule         `bson:"schedules" json:"schedules"`
	DeliveryFee float64            `bson:"delivery_fee" json:"delivery_fee"`
	CreatedAt   string             `bson:"created_at" json:"created_at"`
	UpdatedAt   string             `bson:"updated_at" json:"updated_at"`
}

type PartnerBFFResponse struct {
	Name           string          `json:"name"`
	PartnerID      int             `json:"partner_id"`
	Logo           string          `json:"logo"`
	Schedules      []Schedule      `bson:"schedules" json:"schedules"`
	IsOpen         bool            `json:"is_open"`
	DeliveryFee    float64         `bson:"delivery_fee" json:"delivery_fee"`
	Dishes         []Dish          `json:"dishes"`
	Accompaniments []Accompaniment `json:"accompaniments"`
}

func (h *Handlers) GetBFFByPartnerId(c *gin.Context) {
	partner_id_str := c.Param("partner_id")
	if partner_id_str == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar o id do parceiro como parâmetro")
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

	collectionD := h.database.Collection("Dishes")

	cursor, err := collectionD.Find(h.context, bson.M{})
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}
	dishes := make([]Dish, 0)
	for cursor.Next(h.context) {
		var dish Dish
		if err := cursor.Decode(&dish); err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"status_code": http.StatusInternalServerError,
				"message":     "Não foi possível processar a lista de pratos",
			})
			return
		}
		dishes = append(dishes, dish)
	}

	collectionA := h.database.Collection("Accompaniments")

	cursorA, err := collectionA.Find(h.context, bson.M{})
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}
	accompaniments := make([]Accompaniment, 0)
	for cursorA.Next(h.context) {
		var accompaniment Accompaniment
		if err := cursorA.Decode(&accompaniment); err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"status_code": http.StatusInternalServerError,
				"message":     "Não foi possível processar a lista de pratos",
			})
			return
		}
		accompaniments = append(accompaniments, accompaniment)
	}

	current_time := time.Now()
	partnerBFF := PartnerBFFResponse{
		Name:           partner.Name,
		PartnerID:      partner.PartnerID,
		Logo:           partner.Logo,
		IsOpen:         partner.IsOpen(current_time),
		Schedules:      partner.Schedules,
		DeliveryFee:    partner.DeliveryFee,
		Dishes:         dishes,
		Accompaniments: accompaniments,
	}

	c.IndentedJSON(http.StatusOK, partnerBFF)
}

func (partner *Partner) IsOpen(t time.Time) bool {
	for _, schedule := range partner.Schedules {
		startTime, err := time.Parse("15:04", schedule.StartTime)
		if err != nil {
			return false
		}
		endTime, err := time.Parse("15:04", schedule.EndTime)
		if err != nil {
			return false
		}

		startTime = time.Date(t.Year(), t.Month(), t.Day(), startTime.Hour(), startTime.Minute(), 0, 0, t.Location())
		endTime = time.Date(t.Year(), t.Month(), t.Day(), endTime.Hour(), endTime.Minute(), 0, 0, t.Location())

		if t.Weekday().String() == schedule.DayOfWeek {
			if (t.After(startTime) || t.Equal(startTime)) && (t.Before(endTime) || t.Equal(endTime)) {
				return true
			}
		}
	}
	return false
}
