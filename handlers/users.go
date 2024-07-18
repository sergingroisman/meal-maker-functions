package handlers

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sergingroisman/meal-maker-functions/cmd/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Address struct {
	CEP        string `bson:"cep" json:"cep"`
	Reference  string `bson:"reference" json:"reference"`
	City       string `bson:"city" json:"city"`
	Complement string `bson:"complement" json:"complement"`
	Street     string `bson:"street" json:"street"`
}

type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name        string             `bson:"name" json:"name"`
	PhoneNumber string             `bson:"phone_number" json:"phone_number"`
	Password    string             `bson:"password" json:"password"`
	PartnerID   int                `bson:"partner_id" json:"partner_id"`
	Address     Address            `bson:"address" json:"address"`
	CreatedAt   string             `bson:"created_at" json:"created_at"`
	UpdatedAt   string             `bson:"updated_at" json:"updated_at"`
}

type SignUpReqBody struct {
	Name        string  `bson:"name" json:"name" validate:"required"`
	PhoneNumber string  `bson:"phone_number" json:"phone_number" validate:"required"`
	Password    string  `bson:"password" json:"password" validate:"required,min=6"`
	Address     Address `bson:"address" json:"address"`
}

type SignInReqBody struct {
	PhoneNumber string `bson:"phone_number" json:"phone_number" validate:"required"`
	Password    string `bson:"password" json:"password" validate:"required,min=6"`
}

type UpdatePasswordReqBody struct {
	Password    string `bson:"password" json:"password" validate:"required,min=6"`
	NewPassword string `bson:"new_password" json:"new_password" validate:"required,min=6"`
}

type SignInResponse struct {
	ID          primitive.ObjectID `json:"_id"`
	Accesstoken string             `json:"access_token"`
	Name        string             `json:"name"`
	PhoneNumber string             `json:"phone_number"`
	PartnerID   int                `json:"partner_id"`
	ExpiresIn   time.Duration      `json:"expires_in"`
}

func (h *Handlers) GetUsers(c *gin.Context) {
	collection := h.database.Collection("Users")

	cursor, err := collection.Find(h.context, bson.M{})
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, err)
		return
	}
	users := make([]User, 0)
	for cursor.Next(h.context) {
		var user User
		if err := cursor.Decode(&user); err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"status_code": http.StatusInternalServerError,
				"message":     "Não foi possível processar a lista de usuários",
			})
			return
		}
		users = append(users, user)
	}

	c.IndentedJSON(http.StatusOK, users)
}

func (h *Handlers) GetUserByPhoneNumber(c *gin.Context) {
	var user User
	phone_number := c.Param("phone_number")
	if phone_number == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar o número de telefone como parâmetro")
		return
	}

	collection := h.database.Collection("Users")
	filter := bson.D{{Key: "phone_number", Value: phone_number}}
	err := collection.FindOne(h.context, filter).Decode(&user)

	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusNotFound, gin.H{
			"status_code": http.StatusNotFound,
			"message":     "Não foi possível encontrar esse usuário pelo número de telefone",
		})
		return
	}

	c.IndentedJSON(http.StatusOK, user)
}

func (h *Handlers) SignUp(c *gin.Context) {
	body := SignUpReqBody{}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível criar esse usuário",
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Algo está errado com o body da requisição",
		})
		return
	}

	var user User
	collection := h.database.Collection("Users")
	filter_user_by_phone_number := bson.D{{Key: "phone_number", Value: body.PhoneNumber}}

	err := collection.FindOne(h.context, filter_user_by_phone_number).Decode(&user)
	if err != nil {
		log.Println(err.Error())
	}

	if user.PhoneNumber != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Usuário já foi cadastrado com o mesmo número de telefone",
		})
		return
	}

	password_encoded := passwordEncodeB64(body.Password)
	user = User{
		ID:          primitive.NewObjectID(),
		Name:        body.Name,
		PhoneNumber: body.PhoneNumber,
		Password:    password_encoded,
		PartnerID:   1,
		Address: Address{
			CEP:        body.Address.CEP,
			Reference:  body.Address.Reference,
			City:       body.Address.City,
			Complement: body.Address.Complement,
			Street:     body.Address.Street,
		},
		CreatedAt: time.Now().String(),
		UpdatedAt: time.Now().String(),
	}

	_, err = collection.InsertOne(h.context, user)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"status_code": http.StatusInternalServerError,
			"message":     "Usuário não foi cadastrado com sucesso",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": http.StatusOK,
		"user":        user,
	})
}

func (h *Handlers) SignIn(c *gin.Context) {
	body := SignInReqBody{}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível processar esse número de telefone e senha",
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

	var user User
	filter_user_by_phone_number := bson.D{{Key: "phone_number", Value: body.PhoneNumber}}
	collection := h.database.Collection("Users")

	err := collection.FindOne(h.context, filter_user_by_phone_number).Decode(&user)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível encontrar esse usuário pelo número de telefone",
		})
		return
	}

	if !passwordsMatch(user.Password, body.Password) {
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível realizar o login com essa combinação de senha",
		})
		return
	}

	var bearer_token strings.Builder
	bearer_token.WriteString("Bearer ")
	token, err := createToken(user.PhoneNumber)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível criar um token de acesso",
		})
		return
	}

	bearer_token.WriteString(token)
	res := SignInResponse{
		ID:          user.ID,
		Accesstoken: bearer_token.String(),
		Name:        user.Name,
		PartnerID:   1,
		PhoneNumber: user.PhoneNumber,
		ExpiresIn:   (time.Duration(30*24) * time.Hour),
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": http.StatusOK,
		"loggedIn":    res,
	})
}

func (h *Handlers) UpdatePassword(c *gin.Context) {
	body := UpdatePasswordReqBody{}
	phone_number := c.Param("phone_number")
	if phone_number == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar o número de telefone como parâmetro")
		return
	}

	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível processar esse número de telefone e senha",
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

	var user User
	filter_user_by_phone_number := bson.D{{Key: "phone_number", Value: phone_number}}
	collection := h.database.Collection("Users")
	err := collection.FindOne(h.context, filter_user_by_phone_number).Decode(&user)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível encontrar esse usuário pelo número de telefone",
		})
		return
	}

	if !passwordsMatch(user.Password, body.Password) {
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Não foi possível realizar o login com essa combinação de senha",
		})
		return
	}
	filter := bson.D{{Key: "_id", Value: user.ID}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "password", Value: passwordEncodeB64(body.NewPassword)},
		{Key: "updated_at", Value: time.Now().String()},
	}}}
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
		"status_code": http.StatusOK,
		"message":     "Senha atualizada com sucesso",
	})
}

func verifyToken(token_string string) (*jwt.Token, error) {
	var SECRET = []byte(config.Env.Auth.SecretKey)
	token, err := jwt.Parse(token_string, func(token *jwt.Token) (interface{}, error) {
		return SECRET, nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil
}

func passwordEncodeB64(password string) string {
	password_encoded := base64.StdEncoding.EncodeToString([]byte(password))
	return password_encoded
}

func passwordsMatch(hashed_password, password string) bool {
	password_encoded := passwordEncodeB64(password)
	return hashed_password == password_encoded
}

func createToken(phone_number string) (string, error) {
	var SECRET = []byte(config.Env.Auth.SecretKey)

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": phone_number,
		"iss": "meal-maker-functions",
		"aud": getRole(),
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})

	token_string, err := claims.SignedString(SECRET)
	if err != nil {
		return "", err
	}

	return token_string, nil
}

func getRole() string {
	// if username == "admin" {
	// 	return "admin"
	// }
	return "client"
}
