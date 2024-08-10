package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Dish struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Title       string             `bson:"title" json:"title"`
	Price       float64            `bson:"price" json:"price"`
	Description string             `bson:"description" json:"description,omitempty"`
	Serves      int                `bson:"serves" json:"serves"`
	DayOfWeek   string             `bson:"day_of_week" json:"day_of_week"`
	ImgURL      string             `bson:"img_url" json:"img_url"`
	Active      bool               `bson:"active" json:"active"`
	CreatedAt   string             `bson:"created_at" json:"created_at"`
	UpdatedAt   string             `bson:"updated_at" json:"updated_at"`
}

type TDishReqBody struct {
	Title       string  `bson:"title" json:"title" validate:"required,max=50"`
	Price       float64 `bson:"price" json:"price" validate:"required"`
	Description string  `bson:"description" json:"description" validate:"max=200"`
	Serves      int     `bson:"serves" json:"serves"`
	DayOfWeek   string  `bson:"day_of_week" json:"day_of_week"`
	ImgURL      string  `bson:"img_url" json:"img_url"`
	Active      *bool   `bson:"active" json:"active"`
}

func (h *Handlers) GetDishes(c *gin.Context) {
	collection := h.database.Collection("Dishes")

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

		createdAt, err := parseCreatedAt(dish.CreatedAt)
		if err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"status_code": http.StatusInternalServerError,
				"message":     "Erro ao converter a data de criação",
			})
			return
		}

		dishes = append(dishes, Dish{
			ID:          dish.ID,
			Title:       dish.Title,
			Price:       dish.Price,
			Description: dish.Description,
			Serves:      dish.Serves,
			DayOfWeek:   dish.DayOfWeek,
			ImgURL:      dish.ImgURL,
			Active:      dish.Active,
			CreatedAt:   createdAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   dish.UpdatedAt,
		})
	}

	c.IndentedJSON(http.StatusOK, dishes)
}

func (h *Handlers) GetDishBydId(c *gin.Context) {
	id_str := c.Param("id")
	if id_str == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar o id do prato como parâmetro")
		return
	}
	id, err := primitive.ObjectIDFromHex(id_str)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Formato de ID inválido",
		})
		return
	}

	var dish Dish
	collection := h.database.Collection("Dishes")
	filter := bson.D{{Key: "_id", Value: id}}
	err = collection.FindOne(h.context, filter).Decode(&dish)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusNotFound, gin.H{
			"status_code": http.StatusNotFound,
			"message":     "Não foi possível encontrar um prato com esse ID",
		})
		return
	}

	c.IndentedJSON(http.StatusOK, dish)
}

func (h *Handlers) CreateDish(c *gin.Context) {
	body := TDishReqBody{}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível criar esse prato",
		})
		return
	}

	dish := Dish{
		ID:          primitive.NewObjectID(),
		Title:       body.Title,
		Price:       body.Price,
		Description: body.Description,
		Serves:      body.Serves,
		DayOfWeek:   body.DayOfWeek,
		ImgURL:      body.ImgURL,
		Active:      *body.Active,
		CreatedAt:   time.Now().String(),
		UpdatedAt:   time.Now().String(),
	}

	collection := h.database.Collection("Dishes")
	_, err := collection.InsertOne(h.context, dish)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"status_code": http.StatusInternalServerError,
			"message":     "Prato não foi criado, ocorreu um erro inesperado",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": http.StatusCreated,
		"message":     "Prato criado com sucesso",
	})
}

func (h *Handlers) UploadImage(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file not found"})
		return
	}

	url := "https://mealmakerstorage.blob.core.windows.net/"
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"status_code": http.StatusInternalServerError,
			"message":     "Upload failed",
		})
		return
	}

	client, err := azblob.NewClient(url, credential, nil)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"status_code": http.StatusInternalServerError,
			"message":     "Upload failed",
		})
		return
	}

	containerName := "dish-img-containers"
	// fmt.Printf("Creating a container named %s\n", containerName)
	// _, err = client.CreateContainer(h.context, containerName, nil)
	// if err != nil {
	// 	log.Println(err.Error())
	// 	c.JSON(http.StatusInternalServerError, gin.H{
	// 		"status_code": http.StatusInternalServerError,
	// 		"message":     "Upload failed",
	// 	})
	// 	return
	// }

	uniqueId := uuid.New()
	filename := strings.Replace(uniqueId.String(), "-", "", -1)
	fileExt := strings.Split(file.Filename, ".")[1]
	image := fmt.Sprintf("%s.%s", filename, fileExt)

	// Abrindo o arquivo para leitura
	uploadedFile, err := file.Open()
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"status_code": http.StatusInternalServerError,
			"message":     "Upload failed",
		})
		return
	}
	defer uploadedFile.Close()

	// Lendo o conteúdo do arquivo
	fileBytes, err := io.ReadAll(uploadedFile)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"status_code": http.StatusInternalServerError,
			"message":     "Failed to read content from upload",
		})
		return
	}

	_, err = client.UploadBuffer(h.context, containerName, image, fileBytes, &azblob.UploadBufferOptions{})
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"status_code": http.StatusInternalServerError,
			"message":     "Upload failed",
		})
		return
	}

	blobURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", "mealmakerstorage", containerName, image)

	c.JSON(http.StatusOK, gin.H{
		"status_code": http.StatusCreated,
		"image_url":   blobURL,
	})
}

func (h *Handlers) UpdateDish(c *gin.Context) {
	body := TDishReqBody{}
	id_str := c.Param("id")
	if id_str == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar o id do prato como parâmetro")
		return
	}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível processar esses dados para atualizar os pratos",
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Formulário não está válido",
		})
		return
	}

	id, err := primitive.ObjectIDFromHex(id_str)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Formato de ID inválido",
		})
		return
	}

	collection := h.database.Collection("Dishes")
	filter := bson.D{{Key: "_id", Value: id}}
	updateFields := bson.D{}

	if body.Title != "" {
		updateFields = append(updateFields, bson.E{Key: "title", Value: body.Title})
	}
	if body.Price != 0 {
		updateFields = append(updateFields, bson.E{Key: "price", Value: body.Price})
	}
	if body.Description != "" {
		updateFields = append(updateFields, bson.E{Key: "description", Value: body.Description})
	}
	if body.Serves != 0 {
		updateFields = append(updateFields, bson.E{Key: "serves", Value: body.Serves})
	}
	if body.ImgURL != "" {
		updateFields = append(updateFields, bson.E{Key: "img_url", Value: body.ImgURL})
	}

	if body.Active != nil {
		updateFields = append(updateFields, bson.E{Key: "active", Value: *body.Active})
	}

	updateFields = append(updateFields, bson.E{Key: "updated_at", Value: time.Now().String()})

	update := bson.D{{Key: "$set", Value: updateFields}}

	opts := options.Update().SetUpsert(false)
	_, err = collection.UpdateOne(h.context, filter, update, opts)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível atualizar a senha",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": http.StatusNoContent,
		"message":     "Acompanhamentos atualizados com sucesso",
	})
}

func (h *Handlers) DeleteDish(c *gin.Context) {
	id_str := c.Param("id")
	if id_str == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar o id do acompanhamento como parâmetro")
		return
	}
	id, _ := primitive.ObjectIDFromHex(id_str)

	collection := h.database.Collection("Dishes")
	filter := bson.D{{Key: "_id", Value: id}}

	_, err := collection.DeleteOne(h.context, filter)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível deletar o prato",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": http.StatusOK,
		"message":     "Prato deletado com sucesso",
	})
}
