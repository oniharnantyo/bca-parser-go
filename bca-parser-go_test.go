package bca_parser_go

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func Initiate() {
	config := Config{
		Username: "YOUR_USERNAME",
		Password: "YOUR_PASSWORD",
	}

	Init(config)
}

func TestLogin(t *testing.T) {
	Initiate()
	err := Login(config.Username, config.Password)
	if err != nil {
		fmt.Println(err)
	}
}

func TestGetSaldo(t *testing.T) {
	Initiate()
	saldo, err := GetSaldo()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(saldo)
	Logout()
}

func TestGetMutasiRekening(t *testing.T) {
	Initiate()

	from := time.Now().Add(-10 * 24 * time.Hour)
	to := time.Now()

	mutasiList, err := GetMutasiRekening(from, to)
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println(mutasiList)
	//
	asd, _ := json.Marshal(mutasiList)
	fmt.Println(string(asd))
	Logout()
}

func TestLogout(t *testing.T) {
	Initiate()

	err := Logout()
	if err != nil {
		fmt.Println(err)
	}
}
