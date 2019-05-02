package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	_ "github.com/lib/pq"
)

var (
	host     = os.Getenv("dbhost")
	port     = os.Getenv("dbport")
	user     = os.Getenv("dbuser")
	password = os.Getenv("dbpassword")
	dbname   = os.Getenv("dbname")
)

var (
	accounts     []Account
	lastInsertID int
	db           *sql.DB
)

// Account is used to hold the account details
type Account struct {
	ID           int64  `json:"id,omitempty"`
	FirstName    string `json:"firstname,omitempty"`
	LastName     string `json:"lastname,omitempty"`
	MobileNumber int64  `json:"mobilenumber,omitempty"`
	Password     string `json:"password,omitempty"`
}

type errors struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	Message string
}

// CreateAccount is used to create an account
func CreateAccount(w http.ResponseWriter, r *http.Request) {

	account := &Account{}

	// read request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	// unmarshal body into the object called account
	err = json.Unmarshal(body, account)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// create account row in db
	lastInterstID, err := account.createInDB()
	if err != nil {
		body, _ := json.Marshal(&errors{Code: http.StatusBadRequest, Message: err.Error()})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(body)
		return
	}

	b, _ := json.Marshal(&response{Message: fmt.Sprintf("last insert ID is %v", lastInterstID)})
	w.WriteHeader(http.StatusCreated)
	w.Write(b)
}

// UpdateAccount is used to update the account details
func UpdateAccount(w http.ResponseWriter, r *http.Request) {

	a := &Account{}

	// capture query params
	queryParams := mux.Vars(r)

	// convert id to int64
	id, err := strconv.ParseInt(queryParams["id"], 10, 64)
	if err != nil {
		body, _ := json.Marshal(&errors{Code: http.StatusInternalServerError, Message: err.Error()})
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(body)
		return
	}

	// check if account with given id exists
	account, err := a.getDetailsFromDBbyID(id)
	if err != nil {
		body, _ := json.Marshal(&errors{Code: http.StatusNotFound, Message: err.Error()})
		w.WriteHeader(http.StatusNotFound)
		w.Write(body)
		return
	}

	// if length of an account details is zero, then return response account not found
	if len(account) == 0 {
		body, _ := json.Marshal(&errors{Code: http.StatusNotFound, Message: fmt.Sprintln("Account with id not found")})
		w.WriteHeader(http.StatusNotFound)
		w.Write(body)
		return
	}

	// read request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// unmarshal body into the object called account
	err = json.Unmarshal(body, a)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = a.updateInDB(id)
	if err != nil {
		body, _ := json.Marshal(&errors{Code: http.StatusBadRequest, Message: err.Error()})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(body)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ListAccounts is used to list all the accounts
func ListAccounts(w http.ResponseWriter, r *http.Request) {

	account := &Account{}

	acc, err := account.getDetailsFromDB()
	if err != nil {
		body, _ := json.Marshal(&errors{Code: http.StatusNotFound, Message: err.Error()})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(body)
	}

	resp, err := json.Marshal(acc)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// GetDetailsByID is used to get the details of an account by ID
func GetDetailsByID(w http.ResponseWriter, r *http.Request) {

	queryParams := mux.Vars(r)

	account := &Account{}

	// convert id to int64
	id, err := strconv.ParseInt(queryParams["id"], 10, 64)
	if err != nil {
		body, _ := json.Marshal(&errors{Code: http.StatusInternalServerError, Message: err.Error()})
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(body)
		return
	}

	accs, err := account.getDetailsFromDBbyID(id)
	if err != nil {
		body, _ := json.Marshal(&errors{Code: http.StatusBadRequest, Message: err.Error()})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(body)
		return
	}

	if len(accs) == 0 {
		body, _ := json.Marshal(&errors{Code: http.StatusNotFound, Message: fmt.Sprintln("Account with the ID not exist")})
		w.WriteHeader(http.StatusNotFound)
		w.Write(body)
		return
	}

	body, _ := json.Marshal(accs)
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

// DeleteAccount is used to detete the details of an account by name
func DeleteAccount(w http.ResponseWriter, r *http.Request) {

	a := Account{}

	//capture query params
	queryParams := mux.Vars(r)

	// convert id to int64
	id, err := strconv.ParseInt(queryParams["id"], 10, 64)
	if err != nil {
		body, _ := json.Marshal(&errors{Code: http.StatusInternalServerError, Message: err.Error()})
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(body)
		return
	}

	// delete account in db
	err = a.deleteInDB(id)
	if err != nil {
		body, _ := json.Marshal(&errors{Code: http.StatusBadRequest, Message: err.Error()})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(body)
		return
	}

	// write the response back
	w.WriteHeader(http.StatusNoContent)
}

func (a Account) createInDB() (int, error) {
	sqlStmt := fmt.Sprintf("INSERT INTO accounts_db(firstname,lastname,mobilenumber, password) VALUES('%v','%v','%v','%v') returning uid;", a.FirstName, a.LastName, a.MobileNumber, a.Password)

	err := db.QueryRow(sqlStmt).Scan(&lastInsertID)
	if err != nil {
		return 0, err
	}
	return lastInsertID, nil
}

func (a Account) updateInDB(id int64) error {

	sqlStmt := fmt.Sprintf("update accounts_db set")

	if a.FirstName != "" {
		sqlStmt = fmt.Sprintf("%v firstname='%v',", sqlStmt, a.FirstName)
	}

	if a.LastName != "" {
		sqlStmt = fmt.Sprintf("%v lastname='%v',", sqlStmt, a.LastName)
	}

	if a.MobileNumber != 0 {
		sqlStmt = fmt.Sprintf("%v mobilenumber='%v',", sqlStmt, a.MobileNumber)
	}

	if a.Password != "" {
		sqlStmt = fmt.Sprintf("%v password='%v',", sqlStmt, a.Password)
	}

	// trim the extra comma
	sqlStmt = strings.Trim(sqlStmt, ",")

	// send query to db
	sqlStmt = fmt.Sprintf("%v where uid='%v'", sqlStmt, id)

	_, err := db.Query(sqlStmt)
	return err
}

// getDetailsFromDB will list all the accounts present in DB
func (a Account) getDetailsFromDB() ([]Account, error) {

	var ac = []Account{}

	rows, err := db.Query("SELECT * FROM accounts_db")
	if err != nil {
		fmt.Println(err)
	}

	for rows.Next() {
		var uid int64
		var firstname string
		var lastname string
		var password string
		var mobilenumber int64
		err = rows.Scan(&uid, &firstname, &lastname, &password, &mobilenumber)
		if err != nil {
			fmt.Println(err)
		}

		ac = append(ac, Account{
			FirstName:    firstname,
			LastName:     lastname,
			MobileNumber: mobilenumber,
			Password:     password,
			ID:           uid,
		})

	}

	return ac, nil
}

// getDetailsFromDBbyID is used to get the details of an account by ID
func (a Account) getDetailsFromDBbyID(id int64) ([]Account, error) {

	var ac = []Account{}

	rows, err := db.Query(fmt.Sprintf("SELECT * FROM accounts_db where uid=%v", id))
	if err != nil {
		fmt.Println(err)
	}

	for rows.Next() {
		var uid int64
		var firstname string
		var lastname string
		var password string
		var mobilenumber int64
		err = rows.Scan(&uid, &firstname, &lastname, &password, &mobilenumber)
		if err != nil {
			fmt.Println(err)
		}

		ac = append(ac, Account{
			FirstName:    firstname,
			LastName:     lastname,
			MobileNumber: mobilenumber,
			Password:     password,
			ID:           uid,
		})

	}

	return ac, nil
}

// delete account details in db, that matches id
func (a Account) deleteInDB(id int64) error {
	_, err := db.Query(fmt.Sprintf("delete from accounts_db where uid=%v", id))
	return err
}

func main() {

	var err error

	dbinfo := fmt.Sprintf("host=%s port=%v user=%s "+"password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err = sql.Open("postgres", dbinfo)
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()

	fmt.Println("DB engine started")

	router := mux.NewRouter()

	router.HandleFunc("/account", CreateAccount).Methods(http.MethodPost)        // to create an account
	router.HandleFunc("/account", ListAccounts).Methods(http.MethodGet)          // to get the details of all accounts
	router.HandleFunc("/account/{id}", GetDetailsByID).Methods(http.MethodGet)   // to get the details of specific account
	router.HandleFunc("/account/{id}", DeleteAccount).Methods(http.MethodDelete) // to delete the details of specific account
	router.HandleFunc("/account/{id}", UpdateAccount).Methods(http.MethodPut)    //to update the account details

	fmt.Println("listening started in : 8080")

	http.ListenAndServe(":8080", router)
}
