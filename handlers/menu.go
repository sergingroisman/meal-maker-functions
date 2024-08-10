package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Accompaniment struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Title            string             `bson:"title" json:"title"`
	SmallDescription string             `bson:"small_description" json:"small_description"`
	CreatedAt        string             `bson:"created_at" json:"created_at"`
	UpdatedAt        string             `bson:"updated_at" json:"updated_at"`
}

type Menu struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name             string             `bson:"name" json:"name"`
	SmallDescription string             `bson:"small_description" json:"small_description"`
	PartnerID        int                `bson:"partner_id" json:"partner_id"`
	Dishes           []Dish             `bson:"dishes" json:"dishes"`
	CreatedAt        string             `bson:"created_at" json:"created_at"`
	UpdatedAt        string             `bson:"updated_at" json:"updated_at"`
}

type MenuCreateReqBody struct {
	Name             string   `bson:"name" json:"name" validate:"required,max=40"`
	SmallDescription string   `bson:"small_description" json:"small_description"`
	PartnerID        int      `bson:"partner_id" json:"partner_id"`
	Dishes           []string `bson:"dishes" json:"dishes"`
}

type DishCreateReqBody struct {
	Title                  string   `bson:"title" json:"title" validate:"required,max=40"`
	Price                  float64  `bson:"price" json:"price" validate:"required"`
	MenuID                 string   `bson:"menu_id" json:"menu_id" validate:"required"`
	Discount               float64  `bson:"discount" json:"discount"`
	Description            string   `bson:"description" json:"description"`
	Observation            string   `bson:"observation" json:"observation"`
	MaxAccompanimentsCount int      `bson:"max_accompaniments_count" json:"max_accompaniments_count"`
	Accompaniments         []string `bson:"accompaniments" json:"accompaniments"`
}

type AccompanimentCreateOrUpdate struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Title            string             `bson:"title" json:"title"`
	SmallDescription string             `bson:"small_description" json:"small_description"`
}

type AccompanimentCreateReqBody struct {
	Accompaniments []AccompanimentCreateOrUpdate `bson:"accompaniments" json:"accompaniments"`
}

func (h *Handlers) GetMenusByPartnerId(c *gin.Context) {
	collection := h.database.Collection("Menus")

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

	filter := bson.D{{Key: "partner_id", Value: partner_id}}
	cursor, err := collection.Find(h.context, filter)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}
	menus := make([]Menu, 0)
	for cursor.Next(h.context) {
		var menu Menu
		if err := cursor.Decode(&menu); err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"status_code": http.StatusInternalServerError,
				"message":     "Não foi possível processar a lista de usuários",
			})
			return
		}
		menus = append(menus, menu)
	}

	c.IndentedJSON(http.StatusOK, menus)
}

func (h *Handlers) GetMenuById(c *gin.Context) {
	collection := h.database.Collection("Menus")

	menu_id_str := c.Param("menu_id")
	if menu_id_str == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar id do prato por parâmetro")
		return
	}

	var menu Menu
	menu_id, _ := primitive.ObjectIDFromHex(menu_id_str)
	filter := bson.D{{Key: "_id", Value: menu_id}}
	err := collection.FindOne(h.context, filter).Decode(&menu)

	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusNotFound, gin.H{
			"status_code": http.StatusNotFound,
			"message":     "Não foi possível encontrar um prato com esse ID",
		})
		return
	}

	c.IndentedJSON(http.StatusOK, menu)
}

func (h *Handlers) CreateMenuByPartnerID(c *gin.Context) {
	body := MenuCreateReqBody{}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível criar esse cardápio",
		})
		return
	}

	dishes := make([]Dish, 0)
	if len(body.Dishes) > 0 {
		collectionD := h.database.Collection("Dishes")

		for _, dishID := range body.Dishes {
			dish_id, _ := primitive.ObjectIDFromHex(dishID)
			filter := bson.D{{Key: "_id", Value: dish_id}}
			cursor, err := collectionD.Find(h.context, filter)
			if err != nil {
				c.IndentedJSON(http.StatusInternalServerError, err)
				return
			}

			for cursor.Next(h.context) {
				var dish Dish
				if err := cursor.Decode(&dish); err != nil {
					log.Println(err.Error())
					c.JSON(http.StatusInternalServerError, gin.H{
						"status_code": http.StatusInternalServerError,
						"message":     "Não foi possível processar a lista de acompanhamentos",
					})
					return
				}
				dishes = append(dishes, dish)
			}
		}
	}

	partner_id := 1
	if body.PartnerID != 0 {
		partner_id = body.PartnerID
	}

	menu := Menu{
		ID:               primitive.NewObjectID(),
		Name:             body.Name,
		SmallDescription: body.SmallDescription,
		PartnerID:        partner_id,
		Dishes:           dishes,
		CreatedAt:        time.Now().String(),
		UpdatedAt:        time.Now().String(),
	}

	collection := h.database.Collection("Menus")
	_, err := collection.InsertOne(h.context, menu)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"status_code": http.StatusInternalServerError,
			"message":     "Prato não foi cadastrado com sucesso",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": http.StatusOK,
		"menu":        menu,
	})
}

func (h *Handlers) GetAccompaniments(c *gin.Context) {
	collection := h.database.Collection("Accompaniments")

	cursor, err := collection.Find(h.context, bson.M{})
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}
	accompaniments := make([]Accompaniment, 0)
	for cursor.Next(h.context) {
		var acc Accompaniment
		if err := cursor.Decode(&acc); err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"status_code": http.StatusInternalServerError,
				"message":     "Não foi possível processar a lista de usuários",
			})
			return
		}
		accompaniments = append(accompaniments, acc)
	}

	c.IndentedJSON(http.StatusOK, accompaniments)
}

func (h *Handlers) CreateAccompaniments(c *gin.Context) {
	body := AccompanimentCreateReqBody{}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível criar esse prato",
		})
		return
	}

	accompaniments := []interface{}{}
	for _, acc := range body.Accompaniments {
		accompaniment := Accompaniment{
			ID:               primitive.NewObjectID(),
			Title:            acc.Title,
			SmallDescription: acc.SmallDescription,
			CreatedAt:        time.Now().String(),
			UpdatedAt:        time.Now().String(),
		}
		accompaniments = append(accompaniments, accompaniment)
	}

	collection := h.database.Collection("Accompaniments")
	_, err := collection.InsertMany(h.context, accompaniments)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"status_code": http.StatusInternalServerError,
			"message":     "Acompanhamentos não foram cadastrado com sucesso",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code":    http.StatusOK,
		"accompaniments": accompaniments,
	})
}

func (h *Handlers) UpdateAccompaniments(c *gin.Context) {
	body := AccompanimentCreateReqBody{}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível processar esses dados para atualizar",
		})
		return
	}

	collection := h.database.Collection("Accompaniments")
	var wm []mongo.WriteModel
	for _, acc := range body.Accompaniments {
		filter := bson.D{{Key: "_id", Value: acc.ID}}
		update := bson.D{{Key: "$set", Value: bson.D{
			{Key: "title", Value: acc.Title},
			{Key: "small_description", Value: acc.SmallDescription},
			{Key: "updated_at", Value: time.Now().String()},
		}}}

		wm = append(wm, mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update),
		)
	}

	_, err := collection.BulkWrite(h.context, wm)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível atualizar os acompanhamentos",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": http.StatusOK,
		"message":     "Acompanhamentos atualizados com sucesso",
	})
}

func (h *Handlers) DeleteAccompanimentById(c *gin.Context) {
	accompaniment_id_str := c.Param("accompaniment_id")
	if accompaniment_id_str == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar o id do acompanhamento como parâmetro")
		return
	}
	accompaniment_id, _ := primitive.ObjectIDFromHex(accompaniment_id_str)

	collection := h.database.Collection("Accompaniments")
	filter := bson.D{{Key: "_id", Value: accompaniment_id}}

	_, err := collection.DeleteOne(h.context, filter)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível deletar o acompanhamento",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": http.StatusOK,
		"message":     "Acompanhamento deletado com sucesso",
	})
}
