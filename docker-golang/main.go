package main

import (
    "context"
    "fmt"
    "net/http"
    "log"
    "time"
	"github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "github.com/go-redis/redis/v8"

)

type User struct {
    Email string `json:"email"`
    Password string `json:"password"`

}
var RedisClient *redis.Client

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
    RedisClient = redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "", // No password
        DB:       0,  // Default DB
    })

    ctx := context.Background()
    pong, err := RedisClient.Ping(ctx).Result()
    if err != nil {
        log.Fatal("Error connecting to Redis:", err)
    }
    log.Println("Connected to Redis:", pong)
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

    jsonData, err := bson.MarshalExtJSON(newUser, true, false)
    if err != nil {
        log.Println("Error encoding user for caching:", err)
    } else {
        err = RedisClient.Set(context.Background(), newUser.Email, jsonData, time.Hour).Err()
        if err != nil {
            log.Println("Error caching user:", err)
        }
    }

    // Respond with success message
    c.JSON(200, gin.H{"message": "User created successfully"})
    })

    r.GET("/get",func(c *gin.Context) {
        cachedUsers, err := RedisClient.Get(context.Background(), "users").Result()
        if err == nil {
            var users []User
            err = bson.UnmarshalExtJSON([]byte(cachedUsers), true, &users)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }
            c.JSON(http.StatusOK, users)
            return
        }

    // Find all users in MongoDB
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

        //Cache the users in Redis for 1 hour
        jsonData, err := bson.MarshalExtJSON(users, true, false)
        if err == nil {
            RedisClient.Set(context.Background(), "users", jsonData, time.Hour)
        }

    // Send the users slice in the response
        c.JSON(http.StatusOK, users)
    })
    r.Run(":8081")    
}

