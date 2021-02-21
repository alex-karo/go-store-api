package main

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"strings"
)

type Good struct {
	Id    int    `json:"id"`
	Name  string `json:"name" binding:"required"`
	Price string `json:"price" binding:"required"`
	Count int    `json:"count" binding:"required"`
}

var db *sql.DB

func main() {
	db = setupDB()
	defer db.Close()

	r := setupRouter()

	err := r.Run(":8080")
	if err != nil {
		log.Fatal(err)
	}
}

func setupDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./store.db")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`
		create table if not exists good (
			id integer not null primary key autoincrement, 
			name text not null,
			price decimal(10, 2) not null,
			count integer 
		)
	`)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("insert into good(id, name, price, count) values (1, 'foo', 10, 22), (2, 'bar', 20, 0), (3, 'baz', 30, 2) on conflict(id) do nothing")
	if err != nil {
		panic(err)
	}
	return db
}

func setupRouter() *gin.Engine {
	r := gin.Default()
	authorized := r.Group("/", gin.BasicAuth(gin.Accounts{
		"admin": "password",
	}))

	r.GET("/goods", func(c *gin.Context) {
		goods := getGoods()
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"data":   goods,
		})
	})
	r.GET("/goods/:id", func(c *gin.Context) {
		good, err := getGood(c.Params.ByName("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"status": "error",
				"error":  "not found",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"data":   good,
		})
	})
	authorized.POST("goods", func(c *gin.Context) {
		// Parse JSON
		var good Good
		/*
		Oh no, this validation method throws error "EOF"
		https://github.com/gin-gonic/gin/issues/2502
		 */
		//if err := c.ShouldBindBodyWith(&good, binding.JSON); err != nil {
		//	log.Print(err);
		//	for _, fieldErr := range err.(validator.ValidationErrors) {
		//		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": fmt.Sprint(fieldErr)})
		//		return // exit on first error
		//	}
		//}
		err := c.BindJSON(&good)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": err.Error()})
			return
		}
		stmt, err := db.Prepare(`insert into good(name, price, count) values(?, ?, ?)`)
		if err != nil {
			log.Fatal(err)
		}
		_, err = stmt.Exec(good.Name, good.Price, good.Count)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
			log.Print(err)
		}
		c.JSON(http.StatusCreated, gin.H{"status": "ok"})
	})

	return r
}

func getGoods() []Good {
	var goods []Good
	rows, err := db.Query(`select * from good`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		row := Good{}
		err = rows.Scan(&row.Id, &row.Name, &row.Price, &row.Count)
		if err != nil {
			log.Fatal(err)
		}
		goods = append(goods, row)
	}
	return goods
}

func getGood(id string) (Good, error) {
	var good Good
	stmt, err := db.Prepare(`select * from good where id = ?`)
	if err != nil {
		log.Fatal(err)
	}
	err = stmt.QueryRow(id).Scan(&good.Id, &good.Name, &good.Price, &good.Count)
	if strings.Contains(err.Error(), "no rows") {
		return good, fmt.Errorf("not found")
	} else {
		log.Fatal(err)
	}
	return good, nil
}
