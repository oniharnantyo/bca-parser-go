package bca_parser_go

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/net/publicsuffix"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	loginUrl     = "https://m.klikbca.com/login.jsp"
	loginAction  = "https://m.klikbca.com/authentication.do"
	logoutAction = "https://m.klikbca.com/authentication.do?value(actions)=logout"
	cekSaldoUrl  = "https://m.klikbca.com/balanceinquiry.do"
)

var (
	defaultHeader = []string{
		"GET /login.jsp HTTP/1.1",
		"Host: m.klikbca.com",
		"Connection: keep-alive",
		"Cache-Control: max-age=0",
		"Upgrade-Insecure-Requests: 1",
		"User-Agent: Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.76 Mobile Safari/537.36",
		"Accept:text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Encoding: gzip, deflate, sdch, br",
		"Accept-Language: en-US,en;q=0.8,id;q=0.6,fr;q=0.4",
	}
	config     Config
	client     *http.Client
	IsLoggedIn bool
)

type (
	Config struct {
		Username string
		Password string
	}

	IpAddress struct {
		Ip string `json:"ip"`
	}

	MutasiRekening struct {
		TransactionDate string `json:"transaction_date"`
		TransactionName string `json:"transaction_name"`
		TransferedBy    string `json:"transfered_by"`
		TransferedOn    string `json:"transfered_on"`
		Amount          string `json:"amount"`
		Description     string `json:"description"`
		Type            string `json:"type"`
	}
)

func Init(conf Config) {
	config = Config{
		Username: conf.Username,
		Password: conf.Password,
	}

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}

	client = &http.Client{
		Jar: jar,
	}
}

func GetIPAddress() (IpAddress, error) {
	var ipAddress IpAddress
	client := client
	request, err := http.NewRequest("GET", "http://myjsonip.appspot.com/", nil)
	if err != nil {
		return ipAddress, err
	}

	response, err := client.Do(request)
	if err != nil {
		return ipAddress, err
	}
	defer response.Body.Close()

	err = json.NewDecoder(response.Body).Decode(&ipAddress)
	if err != nil {
		return ipAddress, err
	}

	return ipAddress, nil
}

func setHeader(r *http.Request) {
	r.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 8.0.0; Pixel 2 XL Build/OPD1.170816.004) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.117 Mobile Safari/537.36")
	r.Header.Set("Origin", "https://m.klikbca.com")
	r.Header.Set("Upgrade-Insecure-Requests", "1")
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	r.Header.Set("Sec-Fetch-User", "?1")
}

func Login(username, password string) error {
	resp, err := client.Get(loginUrl)
	if err != nil {
		log.Fatal(err)
	}

	for _, cookie := range resp.Cookies() {
		log.Println("Cookie: ", cookie.Value)
	}

	ip, _ := GetIPAddress()

	asFidRaw := uuid.NewV4()
	asFid := strings.ReplaceAll(asFidRaw.String(), "-", "")

	params := url.Values{}
	params.Set("value(user_id)", username)
	params.Set("value(pswd)", password)
	params.Set("value(Submit)", "LOGIN")
	params.Set("value(actions)", "login")
	params.Set("value(user_ip)", ip.Ip)
	params.Set("value(mobile)", "true")
	params.Set("mobile", "true")
	params.Set("as_fid", asFid)
	params.Set("value(browser_info)", "Mozilla/5.0 (Linux; Android 8.0.0; Pixel 2 XL Build/OPD1.170816.004) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.117 Mobile Safari/537.36")

	req, err := http.NewRequest(http.MethodPost, loginAction, strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}

	setHeader(req)

	req.Header.Set("Referer", loginUrl)

	response, err := client.Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	_, err = ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	IsLoggedIn = true

	return nil
}

func GetSaldo() (decimal.Decimal, error) {
	if !IsLoggedIn {
		Login(config.Username, config.Password)
	}

	req, err := http.NewRequest(http.MethodPost,
		"https://m.klikbca.com/accountstmt.do?value(actions)=menu",
		nil)
	if err != nil {
		return decimal.Zero, err
	}

	setHeader(req)

	req.Header.Set("Referer", loginAction)

	_, err = client.Do(req)
	if err != nil {
		return decimal.Zero, err
	}

	req, err = http.NewRequest(http.MethodPost,
		cekSaldoUrl,
		nil)
	if err != nil {
		return decimal.Zero, err
	}

	setHeader(req)

	req.Header.Set("Referer", "https://m.klikbca.com/accountstmt.do?value(actions)=menu")

	response, err := client.Do(req)
	if err != nil {
		return decimal.Zero, err
	}

	defer response.Body.Close()

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var saldo decimal.Decimal

	doc.Find("span.blue").Each(func(i int, selection *goquery.Selection) {
		selection.Find("tbody").Last().Each(func(i int, selection *goquery.Selection) {
			selection.Find("tr").Last().Each(func(i int, selection *goquery.Selection) {
				saldo, err = decimal.NewFromString(
					strings.ReplaceAll(selection.Find("td").Last().Text(), ",", ""))
				if err != nil {
					err = err
				}
			})
		})
	})

	return saldo, err

}

func GetMutasiRekening(from time.Time, to time.Time) ([]MutasiRekening, error) {
	if !IsLoggedIn {
		Login(config.Username, config.Password)
	}

	req, err := http.NewRequest(http.MethodPost,
		"https://m.klikbca.com/accountstmt.do?value(actions)=menu",
		nil)
	if err != nil {
		return nil, err
	}

	setHeader(req)

	req.Header.Set("Referer", loginAction)

	_, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	req, err = http.NewRequest(http.MethodPost,
		"https://m.klikbca.com/accountstmt.do?value(actions)=acct_stmt",
		nil)
	if err != nil {
		return nil, err
	}

	setHeader(req)

	req.Header.Set("Referer", "https://m.klikbca.com/accountstmt.do?value(actions)=menu")

	_, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	startYear, startMonth, startDate := from.Date()
	endYear, endMonth, endDate := to.Date()

	params := url.Values{}
	params.Set("value(r1)", "1")
	params.Set("value(D1)", "0")
	params.Set("value(submit1)", "Lihat Mutasi Rekening")
	params.Set("value(tDt)", "")
	params.Set("value(fDt)", "")
	params.Set("value(startDt)", strconv.Itoa(startDate))
	params.Set("value(startMt)", strconv.Itoa(int(startMonth)))
	params.Set("value(startYr)", strconv.Itoa(startYear))
	params.Set("value(endDt)", strconv.Itoa(endDate))
	params.Set("value(endMt)", strconv.Itoa(int(endMonth)))
	params.Set("value(endYr)", strconv.Itoa(endYear))

	req, err = http.NewRequest(http.MethodPost, "https://m.klikbca.com/accountstmt.do?value(actions)=acctstmtview", strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}

	setHeader(req)

	req.Header.Set("Referer", "https://m.klikbca.com/accountstmt.do?value(actions)=acct_stmt")

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var mutasiList []MutasiRekening
	doc.Find("span.blue").Each(func(i int, selection *goquery.Selection) {
		element := selection.Find("table").Eq(1).Children()
		element = element.Find("tr").Next()
		element = element.Find("tbody").First()

		element = element.Find("tr").Each(func(i int, selection *goquery.Selection) {
			var mutasi MutasiRekening
			if i != 0 {
				selection.Find("td").Each(func(i int, selection *goquery.Selection) {
					element, _ := selection.Html()
					fmt.Println("element", i, ":::", element)
					switch i {
					case 0:
						mutasi.TransactionDate = strings.ReplaceAll(element, "<nil>", "")
					case 1:
						text := strings.Split(element, "<br/>")
						switch len(text) {
						case 3:
							mutasi.Amount = strings.TrimSpace(text[2])
							mutasi.TransactionName = strings.TrimSpace(text[0])
						case 4:
							mutasi.Amount = strings.TrimSpace(text[3])
							mutasi.TransactionName = strings.TrimSpace(text[0])
						case 6:
							mutasi.Amount = strings.TrimSpace(text[5])
							mutasi.TransferedBy = strings.TrimSpace(text[2])
							mutasi.TransferedOn = strings.TrimSpace(text[3])
						case 7:
							mutasi.Amount = strings.TrimSpace(text[6])
							mutasi.TransferedBy = strings.TrimSpace(text[3])
							mutasi.TransferedOn = strings.TrimSpace(text[4])
						case 8:
							mutasi.Amount = strings.TrimSpace(text[7])
							mutasi.TransactionName = strings.TrimSpace(text[0])
							mutasi.Description = fmt.Sprintf(`%s %s %s`, text[1], text[2], text[5])
						}
					case 2:
						switch strings.TrimSpace(element) {
						case "CR":
							mutasi.Type = "Credit"
						case "DB":
							mutasi.Type = "Debet"
						}
					}
				})
				mutasiList = append(mutasiList, mutasi)
			}
		})
	})

	return mutasiList, nil
}

func Logout() error {
	req, err := http.NewRequest(http.MethodGet,
		logoutAction,
		nil)
	if err != nil {
		return err
	}
	setHeader(req)
	req.Header.Set("Referer", loginAction)

	response, err := client.Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	_, err = ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	return nil

}
