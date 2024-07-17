package handlers

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
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
	PhoneNumber string             `bson:"phoneNumber" json:"phoneNumber"`
	Password    string             `bson:"password" json:"password"`
	PartnerID   int                `bson:"partnerId" json:"partnerId"`
	Address     Address            `bson:"address" json:"address"`
	CreatedAt   string             `bson:"createdAt" json:"createdAt"`
}

type SignUpReqBody struct {
	Name        string  `bson:"name" json:"name" validate:"required"`
	PhoneNumber string  `bson:"phoneNumber" json:"phoneNumber" validate:"required"`
	Password    string  `bson:"password" json:"password" validate:"required,min=6"`
	Address     Address `bson:"address" json:"address"`
}

type SignInReqBody struct {
	PhoneNumber string `bson:"phoneNumber" json:"phoneNumber" validate:"required"`
	Password    string `bson:"password" json:"password" validate:"required,min=6"`
}

type UpdatePasswordReqBody struct {
	Password    string `bson:"password" json:"password" validate:"required,min=6"`
	NewPassword string `bson:"newPassword" json:"newPassword" validate:"required,min=6"`
}

type SignInResponse struct {
	ID          primitive.ObjectID `json:"_id"`
	Accesstoken string             `json:"accessToken"`
	Name        string             `json:"name"`
	PhoneNumber string             `json:"phoneNumber"`
	PartnerID   int                `json:"partnerId"`
	ExpiresIn   time.Duration      `json:"expiresIn"`
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
			log.Fatalf("%s", err)
			c.IndentedJSON(http.StatusInternalServerError, err)
			return
		}
		users = append(users, user)
	}

	c.IndentedJSON(http.StatusOK, users)
}

func (h *Handlers) GetUserByPhoneNumber(c *gin.Context) {
	var user User
	phoneNumber := c.Param("phoneNumber")
	if phoneNumber == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar o número de telefone como parâmetro")
		return
	}

	collection := h.database.Collection("Users")
	filter := bson.D{{Key: "phoneNumber", Value: phoneNumber}}
	err := collection.FindOne(h.context, filter).Decode(&user)

	if err != nil {
		log.Fatalf("Não foi possível encontrar esse usuário pelo número de telefone%s", err)
		c.IndentedJSON(http.StatusBadRequest, err)
		return
	}

	c.IndentedJSON(http.StatusOK, user)
}

func (h *Handlers) SignUp(c *gin.Context) {
	body := SignUpReqBody{}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Não foi possível criar esse usuário",
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Algo está errado com o body da requisição",
		})
		return
	}

	var user User
	collection := h.database.Collection("Users")
	filterUserByPhoneNumber := bson.D{{Key: "phoneNumber", Value: body.PhoneNumber}}

	err := collection.FindOne(h.context, filterUserByPhoneNumber).Decode(&user)
	if err != nil {
		log.Println(err.Error())
	}

	if user.PhoneNumber != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Usuário já foi cadastrado com o mesmo número de telefone",
		})
		return
	}

	passwordEncoded := passwordEncodeB64(body.Password)
	user = User{
		ID:          primitive.NewObjectID(),
		Name:        body.Name,
		PhoneNumber: body.PhoneNumber,
		Password:    passwordEncoded,
		PartnerID:   1,
		Address: Address{
			CEP:        body.Address.CEP,
			Reference:  body.Address.Reference,
			City:       body.Address.City,
			Complement: body.Address.Complement,
			Street:     body.Address.Street,
		},
		CreatedAt: time.Now().String(),
	}

	_, err = collection.InsertOne(h.context, user)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Usuário não foi cadastrado com sucesso",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": http.StatusOK,
		"user": user,
	})
}

func (h *Handlers) SignIn(c *gin.Context) {
	body := SignInReqBody{}
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Não foi possível processar esse número de telefone e senha",
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Formulário não está válido",
		})
		return
	}

	var user User
	filterUserByPhoneNumber := bson.D{{Key: "phoneNumber", Value: body.PhoneNumber}}
	collection := h.database.Collection("Users")

	err := collection.FindOne(h.context, filterUserByPhoneNumber).Decode(&user)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Não foi possível encontrar esse usuário pelo número de telefone",
		})
		return
	}

	if !passwordsMatch(user.Password, body.Password) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Não foi possível realizar o login com essa combinação de senha",
		})
		return
	}

	var bearerToken strings.Builder
	bearerToken.WriteString("Bearer ")
	token, err := createToken(user.PhoneNumber)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Não foi possível criar um token de acesso",
		})
		return
	}

	bearerToken.WriteString(token)
	res := SignInResponse{
		ID:          user.ID,
		Accesstoken: bearerToken.String(),
		Name:        user.Name,
		PartnerID:   1,
		PhoneNumber: user.PhoneNumber,
		ExpiresIn:   (time.Duration(30*24) * time.Hour),
	}

	c.JSON(http.StatusOK, gin.H{
		"code":        http.StatusOK,
		"accessToken": res,
	})
}

func (h *Handlers) UpdatePassword(c *gin.Context) {
	body := UpdatePasswordReqBody{}
	phoneNumber := c.Param("phoneNumber")
	if phoneNumber == "" {
		c.IndentedJSON(http.StatusBadRequest, "Necessário passar o número de telefone como parâmetro")
		return
	}

	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Não foi possível processar esse número de telefone e senha",
		})
		return
	}

	validate := validator.New()
	if err := validate.Struct(&body); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Formulário não está válido",
		})
		return
	}

	var user User
	filterUserByPhoneNumber := bson.D{{Key: "phoneNumber", Value: phoneNumber}}
	collection := h.database.Collection("Users")
	err := collection.FindOne(h.context, filterUserByPhoneNumber).Decode(&user)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Não foi possível encontrar esse usuário pelo número de telefone",
		})
		return
	}

	if !passwordsMatch(user.Password, body.Password) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Não foi possível realizar o login com essa combinação de senha",
		})
		return
	}

	filter := bson.D{{Key: "_id", Value: user.ID}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "password", Value: passwordEncodeB64(body.NewPassword)}}}}
	opts := options.Update().SetUpsert(false)
	_, err = collection.UpdateOne(h.context, filter, update, opts)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Não foi possível atualizar a senha",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Senha atualizada com sucesso",
	})
}

func verifyToken(tokenString string) (*jwt.Token, error) {
	var secretKey = []byte(os.Getenv("JWT_SECRET"))
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
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
	passwordEncoded := base64.StdEncoding.EncodeToString([]byte(password))
	return passwordEncoded
}

func passwordsMatch(hashedPassword, password string) bool {
	passwordEncoded := passwordEncodeB64(password)
	return hashedPassword == passwordEncoded
}

func createToken(phoneNumber string) (string, error) {
	var secretKey = []byte(os.Getenv("JWT_SECRET"))

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": phoneNumber,
		"iss": "meal-maker-functions",
		"aud": getRole(),
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})

	tokenString, err := claims.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func getRole() string {
	// if username == "admin" {
	// 	return "admin"
	// }
	return "client"
}
