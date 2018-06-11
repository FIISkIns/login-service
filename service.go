package main

import (
	"encoding/json"
	"fmt"
	"github.com/huandu/facebook"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

type LoginResponse struct {
	RedirectUrl string `json:"redirectUrl"`
	UserId      string `json:"userId"`
}

func HandleLogin(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	response := &LoginResponse{}
	defer json.NewEncoder(w).Encode(response)

	code := r.URL.Query().Get("code")
	loginError := r.URL.Query().Get("error")

	redirectUri := config.UiPublicUrl + "/login"

	if code == "" && loginError == "" {
		fbUrl, _ := url.Parse("https://www.facebook.com/v3.0/dialog/oauth")

		q := fbUrl.Query()
		q.Set("app_id", config.FacebookAppId)
		q.Set("redirect_uri", redirectUri)
		fbUrl.RawQuery = q.Encode()

		response.RedirectUrl = fbUrl.String()
	} else {
		response.RedirectUrl = config.UiPublicUrl

		if loginError == "" {
			tokenRes, err := facebook.Get("/oauth/access_token", facebook.Params{
				"client_id":     config.FacebookAppId,
				"client_secret": config.FacebookAppSecret,
				"redirect_uri":  redirectUri,
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

			log.Println(me)

			userId := me.Get("id").(string)
			response.UserId = fmt.Sprintf("facebook.%v", userId)
		}
	}
}

func main() {
	initConfig()

	router := httprouter.New()
	router.GET("/login", HandleLogin)

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(config.Port), router))
}
