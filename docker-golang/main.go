package main

import (
    "context"
    "fmt"
    "net/http"
    "log"
	"github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

type User struct {
    Email string `json:"email"`
    Password string `json:"password"`

}

func main() {
    // Set client options
    clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

    // Connect to MongoDB
    client, err := mongo.Connect(context.Background(), clientOptions)
    if err != nil {
        log.Fatal(err)
    }

    // Ping the MongoDB server
    err = client.Ping(context.Background(), nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Connected to MongoDB!")
    collection := client.Database("docker").Collection("users")
	r:=gin.Default();
	r.POST("/user", func(c *gin.Context) {
        var newUser User
    if err := c.BindJSON(&newUser); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Insert the new user into MongoDB
    _, err := collection.InsertOne(context.Background(), newUser)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Respond with success message
    c.JSON(200, gin.H{"message": "User created successfully"})
    })

    r.GET("/get",func(c *gin.Context) {
        cursor, err := collection.Find(context.Background(), bson.M{})
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        defer cursor.Close(context.Background())
    
        // Iterate over the cursor and store users in a slice
        var users []User
        for cursor.Next(context.Background()) {
            var user User
            if err := cursor.Decode(&user); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }
            users = append(users, user)
        }
    
        // Send the users slice in the response
        c.JSON(http.StatusOK, users)
    })
    r.Run(":8081")    
}

