package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/dimfeld/httptreemux"
	"github.com/go-sql-driver/mysql"
	"github.com/huandu/facebook"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var database *sql.DB

type LoginResponse struct {
	RedirectUrl string `json:"redirectUrl"`
	UserId      string `json:"userId"`
}

type UserInfo struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Picture   string `json:"picture"`
	FirstSeen string `json:"firstSeen"`
}

func createTable() {
	query := "CREATE TABLE users (id VARCHAR(100) NOT NULL PRIMARY KEY, name VARCHAR(200) NOT NULL, picture TEXT, first_seen DATETIME DEFAULT CURRENT_TIMESTAMP)"
	_, err := database.Exec(query)
	log.Println("Table creation returned:", err)
}

func HandleLogin(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	err := database.Ping()
	if err != nil {
		log.Fatalln("Database error:", err)
		return
	}

	response := &LoginResponse{}
	w.Header().Set("Content-Type", "application/json")
	defer json.NewEncoder(w).Encode(response)

	code := r.URL.Query().Get("code")
	loginError := r.URL.Query().Get("error")
	returnUrl := r.URL.Query().Get("state")
	if returnUrl == "" {
		returnUrl = r.URL.Query().Get("return_url")
	}

	redirectUrl := config.UiPublicUrl + "/login"

	if code == "" && loginError == "" {
		fbUrl, _ := url.Parse("https://www.facebook.com/v3.0/dialog/oauth")

		q := fbUrl.Query()
		q.Set("app_id", config.FacebookAppId)
		q.Set("redirect_uri", redirectUrl)
		q.Set("state", returnUrl)
		fbUrl.RawQuery = q.Encode()

		response.RedirectUrl = fbUrl.String()
	} else {
		response.RedirectUrl = config.UiPublicUrl

		if loginError == "" {
			tokenRes, err := facebook.Get("/oauth/access_token", facebook.Params{
				"client_id":     config.FacebookAppId,
				"client_secret": config.FacebookAppSecret,
				"redirect_uri":  redirectUrl,
				"code":          code,
			})
			if err != nil {
				log.Println("Error getting access token", err)
				return
			}

			accessToken := tokenRes.Get("access_token")

			me, err := facebook.Get("/me", facebook.Params{
				"access_token": accessToken,
				"fields":       "id,name,picture",
			})
			if err != nil {
				log.Println("Error getting profile information", err)
				return
			}
			userId := fmt.Sprintf("facebook.%v", me.Get("id").(string))
			userName := me.Get("name").(string)

			picture, err := facebook.Get("/me/picture", facebook.Params{
				"access_token": accessToken,
				"type":         "square",
				"height":       400,
				"redirect":     false,
			})
			var userPicture string
			if err != nil {
				log.Println("Error getting profile picture", err)
				userPicture = me.Get("picture").(map[string]interface{})["data"].(map[string]interface{})["url"].(string)
			} else {
				userPicture = picture.Get("data").(map[string]interface{})["url"].(string)
			}

			_, err = database.Exec("INSERT INTO users(id, name, picture, first_seen) VALUES (?, ?, ?, ?) "+
				"ON DUPLICATE KEY UPDATE name = VALUES(name), picture = VALUES(picture)", userId, userName, userPicture, time.Now().UTC())
			if err != nil {
				log.Println("Error saving user information", err)
				return
			}

			response.RedirectUrl = returnUrl
			response.UserId = userId
		}
	}
}

func HandleGetUserInfo(w http.ResponseWriter, r *http.Request, ps map[string]string) {
	err := database.Ping()
	if err != nil {
		log.Fatalln("Database error:", err)
		return
	}

	userId := ps["userId"]
	ret := &UserInfo{}

	rows, err := database.Query("SELECT id, name, picture, first_seen FROM users WHERE id = ?", userId)
	if err != nil {
		log.Println("Database query error:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var firstSeen mysql.NullTime
		err := rows.Scan(&ret.Id, &ret.Name, &ret.Picture, &firstSeen)
		if err != nil {
			log.Println("Database query error:", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		ret.FirstSeen = firstSeen.Time.Format(time.RFC3339)
	}

	if ret.Id != "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ret)
	} else {
		http.NotFound(w, r)
	}
}

func main() {
	initConfig()

	var err error
	database, err = sql.Open("mysql", config.DatabaseUrl)
	if err != nil {
		log.Println("Database Open error:", err)
	}
	createTable()

	router := httptreemux.New()
	router.GET("/login", HandleLogin)
	router.GET("/:userId", HandleGetUserInfo)

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(config.Port), router))
}
