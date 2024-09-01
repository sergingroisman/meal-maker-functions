package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Delivery struct {
	ID          int    `bson:"_id" json:"_id"`
	Name        string `bson:"name" json:"name"`
	PhoneNumber string `bson:"phone_number" json:"phone_number"`
	CreatedAt   string `bson:"created_at" json:"created_at"`
	UpdatedAt   string `bson:"updated_at" json:"updated_at"`
}

type TDeliveryCreateReqBody struct {
	Name        string `bson:"name" json:"name" validate:"required"`
	PhoneNumber string `bson:"phone_number" json:"phone_number" validate:"required"`
}

var (
	deliveryCounter int
	counterMutex    sync.Mutex
)

func (h *Handlers) GetDeliveries(c *gin.Context) {
	collection := h.database.Collection("Deliveries")

	options := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := collection.Find(h.context, bson.M{}, options)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}

	parseCreatedAt := func(dateStr string) (time.Time, error) {
		parts := strings.Split(dateStr, " ")
		datePart := parts[0] // "2024-08-02"
		timePart := parts[1] // "13:58:00.346461603"

		return time.Parse("2006-01-02 15:04:05.999999999", fmt.Sprintf("%s %s", datePart, timePart))
	}

	deliveries := make([]Delivery, 0)
	for cursor.Next(h.context) {
		var delivery Delivery
		if err := cursor.Decode(&delivery); err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"status_code": http.StatusInternalServerError,
				"message":     "Não foi possível processar a lista de entregadores",
			})
			return
		}

		createdAt, err := parseCreatedAt(delivery.CreatedAt)
		if err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"status_code": http.StatusInternalServerError,
				"message":     "Erro ao converter a data de criação",
			})
			return
		}

		deliveries = append(deliveries, Delivery{
			ID:          delivery.ID,
			Name:        delivery.Name,
			PhoneNumber: delivery.PhoneNumber,
			CreatedAt:   createdAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   delivery.UpdatedAt,
		})
	}

	c.IndentedJSON(http.StatusOK, deliveries)
}

func (h *Handlers) CreateDelivery(c *gin.Context) {
	body := TDeliveryCreateReqBody{}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível criar esse entregador",
		})
		return
	}

	counterMutex.Lock()
	defer counterMutex.Unlock()
	deliveryCounter++

	delivery := Delivery{
		ID:          deliveryCounter,
		Name:        body.Name,
		PhoneNumber: body.PhoneNumber,
		CreatedAt:   time.Now().String(),
		UpdatedAt:   time.Now().String(),
	}

	collection := h.database.Collection("Deliveries")
	_, err := collection.InsertOne(h.context, delivery)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"status_code": http.StatusInternalServerError,
			"message":     "Entregador não foi criado, ocorreu um erro inesperado",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": http.StatusCreated,
		"message":     "Entregador criado com sucesso",
	})
}

func (h *Handlers) UpdateDeliveryByID(c *gin.Context) {
	delivery_id_str := c.Param("delivery_id")
	if delivery_id_str == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar o id do entregador como parâmetro")
		return
	}
	delivery_id, err := strconv.Atoi(delivery_id_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Formato de ID inválido",
		})
		return
	}

	body := TDeliveryCreateReqBody{}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível atualizar o entregador",
		})
		return
	}

	collection := h.database.Collection("Orders")
	filter := bson.D{{Key: "_id", Value: delivery_id}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "name", Value: body.Name},
		{Key: "phone_number", Value: body.PhoneNumber},
		{Key: "updated_at", Value: time.Now().String()},
	}}}
	opts := options.Update().SetUpsert(false)
	_, err = collection.UpdateOne(h.context, filter, update, opts)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível atualizar o status do pedido",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": http.StatusOK,
		"message":     "Status do pedido atualizado com sucesso",
	})
}

func (h *Handlers) DeleteDeliveryByID(c *gin.Context) {
	delivery_id_str := c.Param("delivery_id")
	if delivery_id_str == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar o id do entregador como parâmetro")
		return
	}

	deliveryID, err := strconv.Atoi(delivery_id_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Formato de ID inválido",
		})
		return
	}

	collection := h.database.Collection("Deliveries")
	filter := bson.D{{Key: "id", Value: deliveryID}}

	// Execute a operação de delete
	result, err := collection.DeleteOne(h.context, filter)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"status_code": http.StatusInternalServerError,
			"message":     "Não foi possível deletar o entregador",
		})
		return
	}

	// Verifique se algum documento foi deletado
	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"status_code": http.StatusNotFound,
			"message":     "Entregador não encontrado",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": http.StatusOK,
		"message":     "Entregador deletado com sucesso",
	})
}
