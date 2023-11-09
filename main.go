package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const dbname = "fiber-hrms"
const mongoURL = "mongodb://localhost:27017/" + dbname

//in case we want to connect with online mongo server const mongoURL = "mongodb://username@password:localhost:27017"

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

func Connect() error {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURL))
	if err != nil {
		panic(err)
	}
	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := client.Database(dbname)

	mg = MongoInstance{
		Client: client,
		Db:     db,
	}
	return nil
}

func getemployee(c *fiber.Ctx) error {
	query := bson.D{{}}
	cursor, err := mg.Db.Collection("employees").Find(c.Context(), query)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	var employees []Employee
	if err := cursor.All(c.Context(), &employees); err != nil {
		return c.Status(500).SendString(err.Error())
	}

	c.Status(200)
	return c.JSON(employees)
}
func postemployee(c *fiber.Ctx) error {
	var employee Employee
	collection := mg.Db.Collection("employees")

	if err := c.BodyParser(&employee); err != nil {
		c.Status(400)
		return c.JSON(fiber.Map{
			"message": "error while parsing the body",
			"error":   err,
		})
	}

	employee.ID = ""

	insertionresult, err := collection.InsertOne(c.Context(), employee)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	// to double check weather that data has been inserted into the database, we need to m
	filter := bson.D{{Key: "_id", Value: insertionresult.InsertedID}}
	createdRecord := collection.FindOne(c.Context(), filter)
	createdEmployee := &Employee{}
	createdRecord.Decode(createdEmployee)

	return c.Status(201).JSON(createdEmployee)

}
func updateemployee(c *fiber.Ctx) error {
	idparam := c.Params("id")
	employeeid, err := primitive.ObjectIDFromHex(idparam)
	if err != nil {
		return c.SendStatus(400)
	}
	query := bson.D{{Key: "_id", Value: employeeid}}
	var employee Employee
	if err := c.BodyParser(&employee); err != nil {
		return c.Status(400).SendString(err.Error())
	}
	update := bson.D{
		{Key: "$set",
			Value: bson.D{
				{Key: "name", Value: employee.Name},
				{Key: "salary", Value: employee.Salary},
				{Key: "age", Value: employee.Age},
			},
		},
	}

	err = mg.Db.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.SendStatus(404)
		}
		return c.SendStatus(500)
	}
	employee.ID = idparam
	return c.Status(200).JSON(employee)
}

func deleteemployee(c *fiber.Ctx) error {
	idparam := c.Params("id")
	employeeid, err := primitive.ObjectIDFromHex(idparam)
	if err != nil {
		return c.SendStatus(400)
	}
	query := bson.D{{Key: "_id", Value: employeeid}}
	result, err := mg.Db.Collection("employees").DeleteOne(c.Context(), query)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	if result.DeletedCount < 1 {
		return c.SendStatus(404)
	}
	return c.Status(200).JSON("record deleted")
}

func main() {

	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()

	app.Get("/employee", getemployee)
	app.Post("employee", postemployee)
	app.Put("/employee/:id", updateemployee)
	app.Delete("/employee/:id", deleteemployee)

	log.Fatal(app.Listen(":3000"))
}
