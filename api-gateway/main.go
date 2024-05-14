package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type mealType int8
type ingredientType int8
type cuisineType int16

const (
	BREAKFAST mealType = iota
	LUNCH
	DINNER
	DESSERT
	SNACK
)

const (
	VEGETABLE ingredientType = iota
	FRUIT
	MEAT
	MEATALT
	FATS
	DAIRY
	GRAIN
	SPICE
	PULSE
	OTHER
)

type User struct {
	ID           int       `json:"id"`
	Name         *string   `json:"name"`
	Email        *string   `json:"email"`
	Subscription bool      `json:"subscription"`
	CreateDate   time.Time `json:"createdate"`
	LastLogin    time.Time `json:"lastlogin"`
}

type Queue struct {
	ID         int       `json:"id"`
	UserId     int       `json:"userid"`
	RecipeIds  []int     `json:"queue"`
	CreateDate time.Time `json:"createdate"`
	UpdateDate time.Time `json:"updatedate"`
}

type Ingredient struct {
	ID         int            `json:"id"`
	Name       *string        `json:"name"`
	Category   ingredientType `json:"category"`
	CreateDate time.Time      `json:"createdate"`
	UpdateDate time.Time      `json:"updatedate"`
}

type RecipeIngredient struct {
	Ingredient Ingredient     `json:"ingredient"`
	Amount     float32        `json:"amount"`
	SizeAmount ingredientType `json:"sizeamount"`
}

type Recipe struct {
	ID                  int                `json:"id"`
	Name                *string            `json:"name"`
	Instructions        *string            `json:"instructions"`
	Ingredients         []RecipeIngredient `json:"ingredients"`
	Category            mealType           `json:"category"`
	CuisineType         cuisineType        `json:"cuisinetype"`
	Servings            int                `json:"servings"`
	DietaryRestrictions []string           `json:"dietaryrestrictions"`
	TotalTime           time.Duration      `json:"totaltime"`
	PrepTime            time.Duration      `json:"preptime"`
	CookTime            time.Duration      `json:"cooktime"`
	CreatorId           int                `json:"creatorid"`
	CreateDate          time.Time          `json:"createdate"`
	UpdateDate          time.Time          `json:"updatedate"`
}

type RecipePostition struct {
	RecipeId int `json:"recipeid"`
	Position int `json:"position"`
}

var dbPool *pgxpool.Pool
var dbURL = "postgres://apigateway:foodquser@localhost:5432/foodq"

func main() {
	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		panic("Unable to connect to DB")
	}
	defer dbPool.Close()

	err = dbPool.Ping(context.Background())
	if err != nil {
		panic("Could not ping DB")
	}

	router := gin.Default()
	//router.Use(cors.Default())
	router.POST("/internal/users", createUser)
	router.POST("/internal/queues", createQueue)
	router.GET("/recipes", getRecipes)                                         //get all recipes for a specific user
	router.POST("/recipes", addRecipe)                                         //add new recipe to system
	router.GET("/recipes/:recipeid", getRecipe)                                //get specific recipe
	router.PUT("/recipes/:recipeid", updateRecipe)                             //update a specific recipe
	router.GET("/queues/:queueid/recipes", getRecipesInQueue)                  //fetch the recipes in the user's queue
	router.GET("/queues/:queueid/next", getNextRecipeInQueue)                  //fetch the next recipe in the queue based on json data
	router.POST("/queues/:queueid/recipes", addRecipetoQueue)                  //add recipe to users queue
	router.DELETE("/queues/:queueid/recipes/:recipeid", deleteRecipeFromQueue) //remove recipe from users queue
	router.POST("/queues/:queueid/order", updateQueueOrder)                    //update the order of recipes in the queue

	router.Run("localhost:8080")
}

func createUser(c *gin.Context) {
	var newUser User
	if err := c.BindJSON(&newUser); err != nil {
		slog.Error("[createUser] Error unmarshalling JSON", "Error", err, "JSON", c.Request.Body)
		c.Status(http.StatusBadRequest)
		return
	}
	query := `INSERT INTO users (name, email) VALUES (@name, @email)`
	args := pgx.NamedArgs{
		"name":  newUser.Name,
		"email": newUser.Email,
	}
	_, err := dbPool.Exec(context.Background(), query, args)
	if err != nil {
		slog.Error("[createUser] Error adding new user to database", "Error", err, "Query", query, "Args", args)
		c.Status(http.StatusInternalServerError)
	} else {
		c.Status(http.StatusCreated)
	}
}

func createQueue(c *gin.Context) {
	var newQueue Queue
	if err := c.BindJSON(&newQueue); err != nil {
		slog.Error("[createQueue] Error unmarshalling JSON", "Error", err, "JSON", c.Request.Body)
		c.Status(http.StatusBadRequest)
		return
	}
	query := `INSERT INTO queues (userid) VALUES (@userid)`
	args := pgx.NamedArgs{
		"userid": newQueue.UserId,
	}
	_, err := dbPool.Exec(context.Background(), query, args)
	if err != nil {
		slog.Error("[createQueue] Error adding new queue to database", "Error", err, "Query", query, "Args", args)
		c.Status(http.StatusInternalServerError)
	} else {
		c.Status(http.StatusCreated)
	}
}

func getRecipes(c *gin.Context) {
	query := `SELECT * FROM savedrecipes s LEFT JOIN recipes r ON s.recipeid=r.id WHERE s.userid=@userid`
	args := pgx.NamedArgs{
		"userid": c.Query("userid"),
	}
	fmt.Printf("dbPool stats: %v\n", dbPool.Stat())
	rows, err := dbPool.Query(context.Background(), query, args)
	if err != nil {
		slog.Error("[getRecipes] Query failed", "Error", err, "Query", query, "Args", args)
		c.Status(http.StatusBadGateway)
		return
	}
	defer rows.Close()
	recipes, err := pgx.CollectRows(rows, pgx.RowToStructByName[Recipe])
	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Error("[getRecipes] No recipes found for user id", "User Id", c.Query("userid"))
			c.Status(http.StatusNotFound)
		} else {
			slog.Error("[getRecipes] Error unmarshalling rows", "Error", err, "Query", query, "Args", args, "Rows", rows)
			c.Status(http.StatusInternalServerError)
		}
	} else {
		c.JSON(http.StatusOK, recipes)
	}
}

func getRecipe(c *gin.Context) {
	query := `SELECT * FROM recipes WHERE id=@id`
	args := pgx.NamedArgs{
		"id": c.Param("recipeid"),
	}
	rows, err := dbPool.Query(context.Background(), query, args)
	if err != nil {
		slog.Error("[getRecipe] Query failed", "Error", err, "Query", query, "Args", args)
		c.Status(http.StatusBadGateway)
		return
	}
	defer rows.Close()
	recipe, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Recipe])
	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Error("[getRecipe] Recipe not found", "Recipe Id", c.Param("recipeid"))
			c.Status(http.StatusNotFound)
		} else {
			slog.Error("[getRecipe] Error unmarshalling rows", "Error", err, "Query", query, "Args", args, "Rows", rows)
			c.Status(http.StatusInternalServerError)
		}
	} else {
		c.JSON(http.StatusOK, recipe)
	}
}

func addRecipe(c *gin.Context) {
	query := `INSERT INTO recipes (name, instructions, ingredients, category, cuisinetype, servings, dietaryrestrictions, totaltime, preptime, cooktime, creatorid)
		VALUES (@name, @instructions, @ingredients, @category, @cuisinetype, @servings, @dietaryrestrictions, @totaltime, @preptime, @cooktime, @creatorid)`
	var newRecipe Recipe
	if err := c.BindJSON(&newRecipe); err != nil {
		slog.Error("[addRecipe] Error unmarshalling JSON", "Error", err, "JSON", c.Request.Body)
		c.Status(http.StatusBadRequest)
		return
	}
	args := pgx.NamedArgs{
		"name":                newRecipe.Name,
		"instructions":        newRecipe.Instructions,
		"ingredients":         newRecipe.Ingredients,
		"category":            newRecipe.Category,
		"cuisinetype":         newRecipe.CuisineType,
		"servings":            newRecipe.Servings,
		"dietaryrestrictions": newRecipe.DietaryRestrictions,
		"totaltime":           newRecipe.TotalTime,
		"preptime":            newRecipe.PrepTime,
		"cooktime":            newRecipe.CookTime,
		"creatorid":           newRecipe.CreatorId,
	}
	_, err := dbPool.Exec(context.Background(), query, args)
	if err != nil {
		slog.Error("[addRecipe] Error adding recipe to database", "Error", err, "Query", query, "Args", args)
		c.Status(http.StatusInternalServerError)
	} else {
		c.Status(http.StatusCreated)
	}

}

// TODO optional data
func updateRecipe(c *gin.Context) {
	query := `UPDATE recipes SET (name, instructions, ingredients, category, cuisinetype, servings, dietaryrestrictions, totaltime, preptime, cooktime, creatorid, updatedate)
		= (@name, @instructions, @ingredients, @category, @cuisinetype, @servings, @dietaryrestrictions, @totaltime, @preptime, @cooktime, @creatorid, @updatedate) WHERE id=@recipeid`
	var newRecipe Recipe
	if err := c.BindJSON(&newRecipe); err != nil {
		slog.Error("[addRecipe] Error unmarshalling JSON", "Error", err, "JSON", c.Request.Body)
		c.Status(http.StatusBadRequest)
		return
	}
	args := pgx.NamedArgs{
		"name":                newRecipe.Name,
		"instructions":        newRecipe.Instructions,
		"ingredients":         newRecipe.Ingredients,
		"category":            newRecipe.Category,
		"cuisinetype":         newRecipe.CuisineType,
		"servings":            newRecipe.Servings,
		"dietaryrestrictions": newRecipe.DietaryRestrictions,
		"totaltime":           newRecipe.TotalTime,
		"preptime":            newRecipe.PrepTime,
		"cooktime":            newRecipe.CookTime,
		"creatorid":           newRecipe.CreatorId,
		"updatedate":          time.Now().UTC(),
	}
	_, err := dbPool.Exec(context.Background(), query, args)
	if err != nil {
		slog.Error("[addRecipe] Error adding recipe to database", "Error", err, "Query", query, "Args", args)
		c.Status(http.StatusInternalServerError)
	} else {
		c.Status(http.StatusOK)
	}
}

func fetchQueueFromDB(queueId int) (Queue, error) {
	query := `SELECT * FROM queues WHERE queueid=@queueid`
	args := pgx.NamedArgs{
		"queueid": queueId,
	}
	rows, err := dbPool.Query(context.Background(), query, args)
	if err != nil {
		slog.Error("[fetchQueueFromDB] Query failed", "Error", err, "Query", query, "Args", args)
		return Queue{}, err
	}
	defer rows.Close()
	queue, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Queue])
	return queue, err
}

func updateQueueInDB(queueId int, queue []int) error {
	query := `UPDATE queues SET queue = @queue, updatedate = @updatedate WHERE queueid=@queueid`
	args := pgx.NamedArgs{
		"queueid":    queueId,
		"queue":      queue,
		"updatedate": time.Now().UTC(),
	}
	_, err := dbPool.Exec(context.Background(), query, args)
	if err != nil {
		slog.Error("[updateQueueInDB] Update failed", "Error", err, "Query", query, "Args", args)
		return err
	}
	return nil
}

func getRecipesInQueue(c *gin.Context) {
	queueId, err := strconv.Atoi(c.Param("queueid"))
	if err != nil {
		slog.Error("[getRecipesInQueue] Error getting queueid from request string", "Error", err, "Queue Id", c.Param("queueid"))
		c.Status(http.StatusBadRequest)
		return
	}
	queue, err := fetchQueueFromDB(queueId)
	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Error("[getRecipesInQueue] Queue not found", "Queue Id", c.Param("queueid"))
			c.Status(http.StatusNotFound)
		} else {
			slog.Error("[getRecipesInQueue] Internal Error", "Error", err)
			c.Status(http.StatusInternalServerError)
		}
	} else {
		c.JSON(http.StatusOK, queue.RecipeIds)
	}
}

func getNextRecipeInQueue(c *gin.Context) {
	queueId, err := strconv.Atoi(c.Param("queueid"))
	if err != nil {
		slog.Error("[getNextRecipeInQueue] Error getting queueid from request string")
		c.Status(http.StatusBadRequest)
		return
	}
	queue, err := fetchQueueFromDB(queueId)
	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Error("[getNextRecipeInQueue] Queue not found", "Queue Id", c.Param("queueid"))
			c.Status(http.StatusNotFound)
		} else {
			slog.Error("[getNextRecipeInQueue] Internal Error", "Error", err)
			c.Status(http.StatusInternalServerError)
		}
	} else {
		if len(queue.RecipeIds) != 0 {
			c.JSON(http.StatusOK, queue.RecipeIds[0])
		} else {
			slog.Info("[getNextRecipeInQueue] No Recipes in Queue")
			c.JSON(http.StatusOK, queue.RecipeIds)
		}
	}
}

func addRecipetoQueue(c *gin.Context) {
	var recipeId int
	if err := c.BindJSON(recipeId); err != nil {
		slog.Error("[addRecipetoQueue] Error unmarshalling JSON", "Error", err, "JSON", c.Request.Body)
		c.Status(http.StatusBadRequest)
		return
	}
	queueId, err := strconv.Atoi(c.Param("queueid"))
	if err != nil {
		slog.Error("[addRecipetoQueue] Error getting queueid from request string", "Error", err, "Queue Id", c.Param("queueid"))
		c.Status(http.StatusBadRequest)
		return
	}
	queue, err := fetchQueueFromDB(queueId)
	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Error("[addRecipetoQueue] Queue not found", "Queue Id", c.Param("queueid"))
			c.Status(http.StatusNotFound)
		} else {
			slog.Error("[addRecipetoQueue] Internal Error", "Error", err)
			c.Status(http.StatusInternalServerError)
		}
	} else {
		if queue.RecipeIds == nil {
			queue.RecipeIds = []int{recipeId}
		} else {
			queue.RecipeIds = append(queue.RecipeIds, recipeId)
		}
		err := updateQueueInDB(queue.ID, queue.RecipeIds)
		if err != nil {
			slog.Error("[addRecipetoQueue] Error updating queue", "Queue Id", c.Param("queueid"))
			c.Status(http.StatusInternalServerError)
		}
	}
}

func deleteRecipeFromQueue(c *gin.Context) {
	var recipeId int
	if err := c.BindJSON(recipeId); err != nil {
		slog.Error("[deleteRecipeFromQueue] Error unmarshalling JSON", "Error", err, "JSON", c.Request.Body)
		c.Status(http.StatusBadRequest)
		return
	}
	queueId, err := strconv.Atoi(c.Param("queueid"))
	if err != nil {
		slog.Error("[deleteRecipeFromQueue] Error getting queueid from request string", "Error", err, "Queue Id", c.Param("queueid"))
		c.Status(http.StatusBadRequest)
		return
	}
	queue, err := fetchQueueFromDB(queueId)
	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Error("[deleteRecipeFromQueue] Queue not found", "Queue Id", c.Param("queueid"))
			c.Status(http.StatusNotFound)
		} else {
			slog.Error("[deleteRecipeFromQueue] Internal Error", "Error", err)
			c.Status(http.StatusInternalServerError)
		}
		return
	}
	if queue.RecipeIds == nil || len(queue.RecipeIds) == 0 {
		slog.Error("[deleteRecipeFromQueue] Queue empty, cannot delete", "Queue Id", c.Param("queueid"))
		c.Status(http.StatusNotFound)
		return
	}
	newQueue := make([]int, len(queue.RecipeIds)-1)
	found := false
	for _, recipe := range queue.RecipeIds {
		if recipe == recipeId {
			found = true
		} else {
			newQueue = append(newQueue, recipe)
		}
	}
	if found {
		queue.RecipeIds = newQueue
		err := updateQueueInDB(queue.ID, queue.RecipeIds)
		if err != nil {
			slog.Error("[deleteRecipeFromQueue] Error updating queue", "Queue Id", c.Param("queueid"))
			c.Status(http.StatusInternalServerError)
		}
	} else {
		slog.Error("[deleteRecipeFromQueue] Recipe not in queue", "Recipe Id", recipeId, "Queue", queue.RecipeIds)
		c.Status(http.StatusNotFound)
	}
}

// Allow multiple position changes at once?
func updateQueueOrder(c *gin.Context) {
	var recipePostion RecipePostition
	if err := c.BindJSON(recipePostion); err != nil {
		slog.Error("[updateQueueOrder] Error unmarshalling JSON", "Error", err, "JSON", c.Request.Body)
		c.Status(http.StatusBadRequest)
		return
	}
	queueId, err := strconv.Atoi(c.Param("queueid"))
	if err != nil {
		slog.Error("[updateQueueOrder] Error getting queueid from request string", "Error", err, "Queue Id", c.Param("queueid"))
		c.Status(http.StatusBadRequest)
		return
	}
	queue, err := fetchQueueFromDB(queueId)
	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Error("[updateQueueOrder] Queue not found", "Queue Id", c.Param("queueid"))
			c.Status(http.StatusNotFound)
		} else {
			slog.Error("[updateQueueOrder] Internal Error", "Error", err)
			c.Status(http.StatusInternalServerError)
		}
		return
	}
	if queue.RecipeIds == nil || len(queue.RecipeIds) == 0 {
		slog.Error("[updateQueueOrder] Queue empty, cannot delete", "Queue Id", c.Param("queueid"))
		c.Status(http.StatusNotFound)
		return
	}
	newQueue := make([]int, len(queue.RecipeIds))
	found := false
	for index, recipe := range queue.RecipeIds {
		if recipe == recipePostion.RecipeId {
			found = true
		} else if index == recipePostion.Position {
			newQueue = append(newQueue, recipePostion.RecipeId)
		} else {
			newQueue = append(newQueue, recipe)
		}
	}
	if found {
		queue.RecipeIds = newQueue
		err := updateQueueInDB(queue.ID, queue.RecipeIds)
		if err != nil {
			slog.Error("[updateQueueOrder] Error updating queue", "Queue Id", c.Param("queueid"))
			c.Status(http.StatusInternalServerError)
		}
	} else {
		slog.Error("[updateQueueOrder] Recipe not in queue", "Recipe Id", recipePostion, "Queue", queue.RecipeIds)
		c.Status(http.StatusNotFound)
	}
}
