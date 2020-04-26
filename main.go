package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/oauth2.v3"
	"gopkg.in/oauth2.v3/errors"
	"gopkg.in/oauth2.v3/manage"
	"gopkg.in/oauth2.v3/models"
	"gopkg.in/oauth2.v3/server"
	"gopkg.in/oauth2.v3/store"
)

var db *gorm.DB

type User struct {
	ID       string `gorm:"primary_key;not null;unique" json:"id"`
	Username string `gorm:"size:255;not null;unique" json:"username"`
	Email    string `gorm:"size:100;not null;unique" json:"email"`
	Password string `gorm:"size:100;not null;" json:"password"`
}

func Hash(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func (u *User) BeforeSave() error {
	hashedPassword, err := Hash(u.Password)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("sad .env file found")
	}
}

func main() {

	bytes, err := ioutil.ReadFile("seedUsers.json")
	if err != nil {
		log.Fatal(err)
	}

	// Decode
	var users []User
	if err := json.Unmarshal(bytes, &users); err != nil {
		log.Fatal(err)
	}

	err = godotenv.Load()
	if err != nil {
		log.Fatalf("Error getting env, %v", err)
	} else {
		fmt.Println("We are getting the env values")
	}

	//reading database
	dbDriver := os.Getenv("DB_DRIVER")
	dbName := os.Getenv("DB_NAME")
	db, err = gorm.Open(dbDriver, dbName)
	if err != nil {
		fmt.Printf("Cannot connect to %s database\n", dbDriver)
		log.Fatal("This is the error:", err)
	} else {
		fmt.Printf("We are connected to the %s database\n", dbDriver)
	}
	db.Exec("PRAGMA foreign_keys = ON")

	err = db.Debug().DropTableIfExists(&User{}).Error
	if err != nil {
		log.Fatalf("cannot drop table: %v", err)
	}
	err = db.Debug().AutoMigrate(&User{}).Error
	if err != nil {
		log.Fatalf("cannot migrate table: %v", err)
	}

	for i := range users {
		err = db.Debug().Model(&User{}).Create(&users[i]).Error
		if err != nil {
			log.Fatalf("cannot seed users table: %v", err)
		}
		// log.Println(users[i].ID, users[i].Username, users[i].Email, users[i].Password)
	}

	//oauth

	manager := manage.NewDefaultManager()

	cfg := &manage.Config{
		AccessTokenExp:    time.Minute * 5,
		RefreshTokenExp:   time.Hour * 24 * 7,
		IsGenerateRefresh: false,
	}
	manager.SetPasswordTokenCfg(cfg)

	manager.MustTokenStorage(store.NewMemoryTokenStore())

	clientStore := store.NewClientStore()

	AppID := os.Getenv("APP_ID")
	AppSecret := os.Getenv("APP_SECRET")

	for _, p := range users {
		clientStore.Set(AppID, &models.Client{ID: p.ID, Secret: AppSecret})
	}

	manager.MapClientStorage(clientStore)

	srv := server.NewDefaultServer(manager)
	srv.SetAllowGetAccessRequest(true)
	srv.SetAllowedGrantType(oauth2.PasswordCredentials)

	srv.SetClientInfoHandler(server.ClientFormHandler)

	srv.SetPasswordAuthorizationHandler(func(username, password string) (userID string, err error) {
		userID = ""
		user := User{}
		err = db.Debug().Model(User{}).Where("email = ?", username).Take(&user).Error
		if err != nil {
			return "", err
		}
		err = VerifyPassword(user.Password, password)
		if err != nil && err == bcrypt.ErrMismatchedHashAndPassword {
			return "", err
		}
		userID = user.ID
		return
	})

	srv.SetInternalErrorHandler(func(err error) (re *errors.Response) {
		log.Println("Internal Error:", err.Error())
		return
	})

	srv.SetResponseErrorHandler(func(re *errors.Response) {
		log.Println("Response Error:", re.Error.Error())
	})

	http.HandleFunc("/oauth", func(w http.ResponseWriter, r *http.Request) {
		srv.HandleTokenRequest(w, r)
	})

	http.HandleFunc("/home", func(w http.ResponseWriter, r *http.Request) {
		token, err := srv.ValidationBearerToken(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		user := User{}
		db.Debug().Model(User{}).Where("id = ?", token.GetUserID()).Take(&user)

		e := json.NewEncoder(w)
		e.SetIndent("", "  ")
		e.Encode(user)

	})

	fmt.Println("Listening to port 8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
