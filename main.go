package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/crypto/bcrypt"

	"github.com/coopernurse/gorp"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessionauth"
	"github.com/martini-contrib/sessions"
	_ "github.com/mattn/go-sqlite3"
)

type MyGorpTracer struct{}

func (t *MyGorpTracer) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

var dbmap *gorp.DbMap

func initDb() *gorp.DbMap {
	dbName := "martini_app.db"
	_, err := os.Open(dbName)
	if err == nil {
		os.Remove(dbName)
	}

	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatalln("Fail to create database", err)
	}

	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}

	tracer := &MyGorpTracer{}
	dbmap.TraceOn("[SQL trace]", tracer)

	dbmap.AddTableWithName(MyUserModel{}, "users").SetKeys(true, "Id")
	err = dbmap.CreateTablesIfNotExists()
	if err != nil {
		log.Fatalln("Could not build tables", err)
	}

	return dbmap
}

func main() {
	store := sessions.NewCookieStore([]byte("secret123"))
	dbmap = initDb()

	m := martini.Classic()
	m.Use(render.Renderer())

	store.Options(sessions.Options{
		MaxAge: 0,
	})
	m.Use(sessions.Sessions("my_session", store))
	m.Use(sessionauth.SessionUser(GenerateAnonymousUser))
	sessionauth.RedirectUrl = "/new-login"
	sessionauth.RedirectParam = "new-next"

	m.Get("/", func(r render.Render) {
		r.HTML(200, "index", nil)
	})

	m.Get("/new-login", func(r render.Render) {
		r.HTML(200, "login", nil)
	})

	m.Get("/register", func(r render.Render) {
		r.HTML(200, "register", nil)
	})

	m.Post("/register", binding.Bind(MyUserModel{}), func(session sessions.Session, r render.Render, postedUser MyUserModel) {
		user := MyUserModel{Username: postedUser.Username,
			Password:      toHash(postedUser.Password),
			authenticated: false}
		err := dbmap.Insert(&user)
		if err != nil {
			log.Fatalln("Could not insert test user", err)
		}

		err = sessionauth.AuthenticateSession(session, &user)
		if err != nil {
			r.JSON(500, err)
		}

		r.Redirect("/private")
	})

	m.Post("/new-login", binding.Bind(MyUserModel{}),
		func(session sessions.Session, postedUser MyUserModel, r render.Render, req *http.Request) {

			user := MyUserModel{}
			err := dbmap.SelectOne(&user, "SELECT * FROM users WHERE username = $1", postedUser.Username)
			if err != nil {
				fmt.Println("Not found user.")
			}

			err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(postedUser.Password))
			if err != nil {
				r.Redirect(sessionauth.RedirectUrl)
				return
			} else {
				err := sessionauth.AuthenticateSession(session, &user)
				if err != nil {
					r.JSON(500, err)
				}

				params := req.URL.Query()
				redirect := params.Get(sessionauth.RedirectParam)
				r.Redirect(redirect)
				return
			}
		})

	m.Get("/private", sessionauth.LoginRequired,
		func(r render.Render, user sessionauth.User) {
			r.HTML(200, "private", user.(*MyUserModel))
		})

	m.Get("/logout", sessionauth.LoginRequired,
		func(session sessions.Session, user sessionauth.User, r render.Render) {
			sessionauth.Logout(session, user)
			r.Redirect("/")
		})

	m.Run()
}

func toHash(pass string) string {
	converted, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	return string(converted)
}
