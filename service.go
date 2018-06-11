package main

import (
	"github.com/huandu/facebook"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

func HandleLogin(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	code := r.URL.Query().Get("code")
	loginError := r.URL.Query().Get("error")

	redirectUri := config.UiPublicUrl + "/login"

	if code == "" && loginError == "" {
		fbUrl, _ := url.Parse("https://www.facebook.com/v3.0/dialog/oauth")

		q := fbUrl.Query()
		q.Set("app_id", config.FacebookAppId)
		q.Set("redirect_uri", redirectUri)
		fbUrl.RawQuery = q.Encode()

		http.Redirect(w, r, fbUrl.String(), http.StatusFound)
	} else {
		defer http.Redirect(w, r, config.UiPublicUrl, http.StatusFound)

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
		}

	}
}

func main() {
	initConfig()

	router := httprouter.New()
	router.GET("/login", HandleLogin)

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(config.Port), router))
}
