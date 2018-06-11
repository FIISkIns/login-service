package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/dimfeld/httptreemux"
	_ "github.com/go-sql-driver/mysql"
	"github.com/huandu/facebook"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

var database *sql.DB

type LoginResponse struct {
	RedirectUrl string `json:"redirectUrl"`
	UserId      string `json:"userId"`
}

type UserInfo struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func createTable() {
	query := "CREATE TABLE users (id VARCHAR(100) NOT NULL PRIMARY KEY, name VARCHAR(200) NOT NULL, picture TEXT)"
	_, err := database.Exec(query)
	log.Println("Table creation returned:", err)
}

func HandleLogin(w http.ResponseWriter, r *http.Request, _ map[string]string) {
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
			userPicture := me.Get("picture").(map[string]interface{})["data"].(map[string]interface{})["url"].(string)

			_, err = database.Exec("INSERT INTO users(id, name, picture) VALUES (?, ?, ?) "+
				"ON DUPLICATE KEY UPDATE name = VALUES(name), picture = VALUES(picture)", userId, userName, userPicture)
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
	userId := ps["userId"]
	ret := &UserInfo{}

	rows, err := database.Query("SELECT id, name, picture FROM users WHERE id = ?", userId)
	if err != nil {
		log.Println("Database query error:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.Id, &ret.Name, &ret.Picture)
		if err != nil {
			log.Println("Database query error:", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
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
