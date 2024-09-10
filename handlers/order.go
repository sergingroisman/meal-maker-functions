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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OrderStatus int

const (
	OrderSent           OrderStatus = iota // 0
	OrderConfirmed                         // 1
	OrderOutForDelivery                    // 2
	OrderDelivered                         // 3
)

func (os OrderStatus) String() string {
	return [...]string{
		"Pedido Enviado",
		"Pedido Confirmado",
		"Pedido Saiu para Entrega",
		"Pedido Entregue",
	}[os]
}

type OrderDishes struct {
	ID             string          `json:"_id"`
	Title          string          `json:"title"`
	Price          float64         `json:"price"`
	Observation    string          `json:"observation"`
	Quantity       int             `json:"quantity"`
	Accompaniments []Accompaniment `json:"accompaniments"`
}

type Order struct {
	ID            int           `bson:"_id,omitempty" json:"_id,omitempty"`
	User          User          `bson:"user" json:"user"`
	PartnerID     int           `bson:"partner_id" json:"partner_id"`
	Dishes        []OrderDishes `bson:"dishes" json:"dishes"`
	Status        OrderStatus   `bson:"status" json:"status"`
	PaymentType   string        `bson:"payment_type" json:"payment_type"`
	DeliveryID    int           `bson:"delivery_id" json:"delivery_id"`
	DeliveryType  string        `bson:"delivery_type" json:"delivery_type"`
	QuantityTotal int           `bson:"quantity_total" json:"quantity_total"`
	Total         float64       `bson:"total" json:"total"`
	CreatedAt     string        `bson:"created_at" json:"created_at"`
	UpdatedAt     string        `bson:"updated_at" json:"updated_at"`
}

type OrderCreateReqBody struct {
	PartnerID     int           `bson:"partner_id" json:"partner_id"`
	UserID        string        `bson:"user_id" json:"user_id"`
	QuantityTotal int           `bson:"quantity_total" json:"quantity_total"`
	Total         float64       `bson:"total" json:"total"`
	Dishes        []OrderDishes `bson:"dishes" json:"dishes"`
	PaymentType   string        `bson:"payment_type" json:"payment_type"`
	DeliveryType  string        `bson:"delivery_type" json:"delivery_type"`
}

type OrderUpdateReqBody struct {
	Status OrderStatus `bson:"status" json:"status"`
}

type OrderResponse struct {
	ID            int           `json:"_id,omitempty"`
	User          User          `json:"user"`
	PartnerID     int           `json:"partner_id"`
	Dishes        []OrderDishes `json:"dishes"`
	Status        string        `json:"status"`
	PaymentType   string        `json:"payment_type"`
	Delivery      Delivery      `json:"delivery"`
	DeliveryType  string        `json:"delivery_type"`
	QuantityTotal int           `json:"quantity_total"`
	Total         float64       `json:"total"`
	CreatedAt     string        `bson:"created_at" json:"created_at"`
}

var (
	orderCounter  int
	counterOMutex sync.Mutex
)

func (h *Handlers) GetOrdersByPartnerID(c *gin.Context) {
	collection := h.database.Collection("Orders")

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

	parseCreatedAt := func(dateStr string) (time.Time, error) {
		parts := strings.Split(dateStr, " ")
		datePart := parts[0] // "2024-08-02"
		timePart := parts[1] // "13:58:00.346461603"

		return time.Parse("2006-01-02 15:04:05.999999999", fmt.Sprintf("%s %s", datePart, timePart))
	}

	// Calcular o início e fim do dia atual no formato correto
	startOfDay := time.Now().Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)

	startOfDayStr := startOfDay.Format("2006-01-02 15:04:05.999999999 -0300 -03")
	endOfDayStr := endOfDay.Format("2006-01-02 15:04:05.999999999 -0300 -03")

	filter := bson.D{
		{Key: "partner_id", Value: partner_id},
		{Key: "created_at", Value: bson.D{{Key: "$gte", Value: startOfDayStr}, {Key: "$lt", Value: endOfDayStr}}},
	}

	if _, exists := c.GetQuery("feed"); exists {
		filter = append(filter, bson.E{Key: "status", Value: bson.D{{Key: "$ne", Value: OrderDelivered}}})
	}

	options := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := collection.Find(h.context, filter, options)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}
	defer cursor.Close(h.context)

	orders := make([]OrderResponse, 0)
	for cursor.Next(h.context) {
		var order Order
		if err := cursor.Decode(&order); err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"status_code": http.StatusInternalServerError,
				"message":     "Não foi possível processar a lista de pedidos",
			})
			return
		}

		createdAt, err := parseCreatedAt(order.CreatedAt)
		if err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"status_code": http.StatusInternalServerError,
				"message":     "Erro ao converter a data de criação",
			})
			return
		}

		var delivery Delivery
		if order.DeliveryID != 0 {
			collectionD := h.database.Collection("Deliveries")
			filter := bson.D{{Key: "_id", Value: order.DeliveryID}}

			// Realiza a consulta no banco de dados
			err := collectionD.FindOne(h.context, filter).Decode(&delivery)

			if err != nil {
				// Loga o erro e retorna uma resposta JSON apropriada
				log.Println("Erro ao buscar entrega:", err.Error())
				c.JSON(http.StatusNotFound, gin.H{
					"status_code": http.StatusNotFound,
					"message":     "Não foi possível encontrar a entrega com o ID fornecido.",
				})
				return
			}
		}

		orders = append(orders, OrderResponse{
			ID:            order.ID,
			User:          order.User,
			PartnerID:     order.PartnerID,
			Dishes:        order.Dishes,
			Status:        order.Status.String(),
			PaymentType:   order.PaymentType,
			Delivery:      delivery,
			DeliveryType:  order.DeliveryType,
			QuantityTotal: order.QuantityTotal,
			Total:         order.Total,
			CreatedAt:     createdAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.IndentedJSON(http.StatusOK, orders)
}

func (h *Handlers) GetOrdersByUserID(c *gin.Context) {
	collection := h.database.Collection("Orders")

	user_id_str := c.Param("user_id")
	if user_id_str == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar id do cardápio por parâmetro")
		return
	}
	user_id, err := primitive.ObjectIDFromHex(user_id_str)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Formato de ID inválido",
		})
		return
	}

	parseCreatedAt := func(dateStr string) (time.Time, error) {
		parts := strings.Split(dateStr, " ")
		datePart := parts[0] // "2024-08-02"
		timePart := parts[1] // "13:58:00.346461603"

		return time.Parse("2006-01-02 15:04:05.999999999", fmt.Sprintf("%s %s", datePart, timePart))
	}

	filter := bson.D{
		{Key: "user._id", Value: user_id},
	}
	options := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := collection.Find(h.context, filter, options)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}
	defer cursor.Close(h.context)

	orders := make([]OrderResponse, 0)
	for cursor.Next(h.context) {
		var order Order
		if err := cursor.Decode(&order); err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"status_code": http.StatusInternalServerError,
				"message":     "Não foi possível processar a lista de pedidos",
			})
			return
		}

		createdAt, err := parseCreatedAt(order.CreatedAt)
		if err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"status_code": http.StatusInternalServerError,
				"message":     "Erro ao converter a data de criação",
			})
			return
		}

		orders = append(orders, OrderResponse{
			ID:            order.ID,
			User:          order.User,
			PartnerID:     order.PartnerID,
			Dishes:        order.Dishes,
			Status:        order.Status.String(),
			PaymentType:   order.PaymentType,
			DeliveryType:  order.DeliveryType,
			QuantityTotal: order.QuantityTotal,
			Total:         order.Total,
			CreatedAt:     createdAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.IndentedJSON(http.StatusOK, orders)
}

func (h *Handlers) CreateOrderByUser(c *gin.Context) {
	user_id_str := c.Param("user_id")
	if user_id_str == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar o id do usuário por parâmetro")
		return
	}

	counterOMutex.Lock()
	defer counterOMutex.Unlock()
	orderCounter++

	user_id, _ := primitive.ObjectIDFromHex(user_id_str)

	body := OrderCreateReqBody{}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível realizar o pedido",
		})
		return
	}

	partner_id := 1
	if body.PartnerID != 0 {
		partner_id = body.PartnerID
	}

	var user User
	collectionU := h.database.Collection("Users")
	filter := bson.D{{Key: "_id", Value: user_id}}
	err := collectionU.FindOne(h.context, filter).Decode(&user)

	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusNotFound, gin.H{
			"status_code": http.StatusNotFound,
			"message":     "Não foi possível encontrar esse usuário pelo número de telefone",
		})
		return
	}

	var status OrderStatus = OrderSent

	order := Order{
		ID:            orderCounter,
		User:          user,
		Status:        status,
		PartnerID:     partner_id,
		Dishes:        body.Dishes,
		PaymentType:   body.PaymentType,
		DeliveryType:  body.DeliveryType,
		Total:         body.Total,
		QuantityTotal: body.QuantityTotal,
		CreatedAt:     time.Now().String(),
		UpdatedAt:     time.Now().String(),
	}

	collectionO := h.database.Collection("Orders")
	_, err = collectionO.InsertOne(h.context, order)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"status_code": http.StatusInternalServerError,
			"message":     "Pedido não foi efetuado com sucesso",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": http.StatusOK,
		"order":       order,
	})
}

func (h *Handlers) UpdateOrderByUser(c *gin.Context) {
	order_id_str := c.Param("order_id")
	if order_id_str == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar o id do pedido como parâmetro")
		return
	}

	order_id, err := strconv.Atoi(order_id_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Formato de ID inválido",
		})
		return
	}

	body := OrderUpdateReqBody{}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível realizar o pedido",
		})
		return
	}

	var status OrderStatus = body.Status

	collection := h.database.Collection("Orders")
	filter := bson.D{{Key: "_id", Value: order_id}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "status", Value: status},
		{Key: "updated_at", Value: time.Now().String()},
	}}}

	delivery_id_str, deliveryExists := c.GetQuery("delivery_id")

	if deliveryExists {
		delivery_id, err := strconv.Atoi(delivery_id_str)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status_code": http.StatusBadRequest,
				"message":     "Formato de ID de entregador inválido",
			})
			return
		}
		update = append(update, bson.E{
			Key: "$set", Value: bson.D{
				{Key: "delivery_id", Value: delivery_id},
			},
		})
	}
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
