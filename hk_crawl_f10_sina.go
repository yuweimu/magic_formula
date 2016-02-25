package main

import (
	"fmt"
	"strings"
	"strconv"
	"io/ioutil"
  iconv "github.com/djimenez/iconv-go"
	_ "github.com/go-sql-driver/mysql"
	"database/sql"
	"time"
	"errors"
	"net/http"
	"math/rand"
  "encoding/json"
)

//http://quotes.money.163.com/hkstock/cwsj_03333.html
//http://hq.sinajs.cn/list=hk03333

func WriteDB(stockCode, year string, revenue, grossProfit, profit, bookValue, divdend float64,
        currency string, shareCount int64) error {
    db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/magic_formula")
    defer db.Close()
    if err != nil {
        fmt.Printf("WriteDB sql.Open err %s", err.Error())
        return err
    }

    statement, err := db.Prepare("INSERT INTO hk_stock_f10_sina SET stock_code=?," +
                                 "fiscal_year=?,revenue=?,gross_profit=?,earning=?,book_value=?," +
                                 "divdend=?,currency=?,share_count=?")
    if err != nil {
        fmt.Printf("WriteDB db.Prepare err %s", err.Error())
        return err
    }

    res, err := statement.Exec(stockCode, year, revenue, grossProfit,
                               profit, bookValue, divdend, currency, shareCount)
    if err != nil {
        fmt.Printf("WriteDB db.Exec err %s", err.Error())
        return err
    }

    insertId, err := res.LastInsertId()
    if err != nil {
        fmt.Printf("WriteDB err %s", err.Error())
        return err
    }
    fmt.Printf("WriteDB ok, stockCode=%s,insertId=%v", stockCode, insertId)
    return nil
}

var YearList = []string{"2014", "2013", "2012", "2011", "2010", "2009", "2008", "2007", "2006", "2005", "2004"}

func ParseF10Field(list[]string, offset int, hasMore *bool) float64 {
    if offset < len(list) {
        f, err := strconv.ParseFloat(list[offset], 32)
        if err == nil {
		        *hasMore = true
		        return f
        }
    }
    return 0.0
}
func FetchSinaF10Array(code, table, url string) (*[]interface{}, error){
    body, err := httpGet(url)
    if err != nil {
        fmt.Printf("FetchSinaF10Array code=%s table=%s httpGet err %s\r\n",
                   code, table, err.Error())
        return nil, err
    }
    utf8, _ := iconv.ConvertString(body, "gb2312", "utf-8")

    utf8 = strings.Replace(utf8, "var tableData = ([", "[", 1)
    utf8 = strings.Replace(utf8, "]);", "]", 1)

    // fmt.Println(utf8)
	  var list[]interface{}
	  if err := json.Unmarshal([]byte(utf8), &list); err != nil {
        fmt.Printf("FetchSinaF10Array code=%s table=%s format err %s\r\n",
                   code, table, err.Error())
        return nil ,err
	  }
	  fmt.Printf("FetchSinaF10Array code=%s table=%s ok,%s %v\r\n", code, table, list)
	  return &list, nil
}

type AssetInfo struct {
	bookValue float64
	shareCount int64
}

func ParseFloatField(code, year, field string, yearly_data []interface{}, pos int) float64 {
    if yearly_data[pos] == nil {
        fmt.Printf("ParseFloatField nil err, code=%s year=%s field=%s pos=%d\r\n",
                   code, year, field, pos)
        return 0.0
    }
    value, err := strconv.ParseFloat(yearly_data[pos].(string), 32)
    if err != nil {
        fmt.Printf("ParseFloatField format err, code=%s year=%s field=%s pos=%d\r\n",
                   code, year, field, pos)
        return 0.00001
    }
    return value
}

func CrawlF10(code string) {
    var assetMap = make(map[string]*AssetInfo)

    list, err := FetchSinaF10Array(code, "asset", GetUrlF10Asset(code))
    if err != nil {
        fmt.Printf("FetchSinaF10Array asset err %s\r\n", err.Error())
        return
    }

    for i, _ := range *list {
       yearly_data := (*list)[i].([]interface{})
	     fmt.Println(yearly_data)
	     if len(yearly_data) < 27 {
           continue
       }
       var asset AssetInfo
       year := yearly_data[0].(string)
       asset.bookValue = ParseFloatField(code, year, "bookValue", yearly_data, 11)

       if yearly_data[25] != nil {
           strShareCount := strings.Replace(yearly_data[25].(string), "股", "", 1)
           asset.shareCount, _ = strconv.ParseInt(strShareCount, 10, 64)
           fmt.Printf("===============%s\r\n", strShareCount)
       } else {
           fmt.Printf("ParseField shareCount err, code=%s year=%s pos=25\r\n",
                       code, year)
       }
       assetMap[year] = &asset
    }

    list, err = FetchSinaF10Array(code, "earning", GetUrlF10Earning(code))
    if err != nil {
        fmt.Printf("FetchSinaF10Array earning err %s\r\n", err.Error())
        return
    }

    for i, _ := range *list {

	//year := ""
  //revenue := 0.0
  //profit := 0.0
  //grossProfit := 0.0
  //divdend := 0.0
  //bookValue := 0.0
  //var shareCount int64 = 0
  //var currency string
       yearly_data := (*list)[i].([]interface{})
	     fmt.Println(yearly_data)
	     if len(yearly_data) < 22 {
           fmt.Printf("earning too few fields\r\n")
           continue
       }
       year := yearly_data[0].(string)
       asset, ok := assetMap[year]
       if !ok {
           fmt.Printf("earning %s no asset err.\r\n", year)
           continue
       }
       // revenue, _ := strconv.ParseFloat(yearly_data[2].(string), 32)
       revenue := ParseFloatField(code, year, "revenue", yearly_data, 2)

       // grossProfit, _ := strconv.ParseFloat(yearly_data[18].(string), 32)
       grossProfit := ParseFloatField(code, year, "grossProfit", yearly_data, 18)

       // profit, _ := strconv.ParseFloat(yearly_data[7].(string), 32)
       profit := ParseFloatField(code, year, "profit", yearly_data, 7)

       // divdend, _ := strconv.ParseFloat(yearly_data[8].(string), 32)
       divdend := ParseFloatField(code, year, "divdend", yearly_data, 8)

       currency := yearly_data[21].(string)

       fmt.Printf("Year %s ------%4.3f,%4.3f,%4.3f,%4.3f,%4.3f,%s,%d\r\n",
                  year, profit, revenue, divdend, grossProfit,
                  asset.bookValue, currency, asset.shareCount)
       WriteDB(code, year, revenue, grossProfit, profit, asset.bookValue,
               divdend, currency, asset.shareCount)
    }
}

func FullCode(code string) string {
	  return "hk" + code
}
func GetUrlF10Asset(code string) string {
	return "http://stock.finance.sina.com.cn/hkstock/api/jsonp.php/var%20tableData%20=%20" + 
	       "/FinanceStatusService.getBalanceSheetForjs?symbol=" + code + "&balanceSheet=zero"
}

func GetUrlF10Earning(code string) string {
  // return "http://quotes.money.163.com/hkstock/cwsj_" + code + ".html"
  return "http://stock.finance.sina.com.cn/hkstock/api/jsonp.php/var%20tableData%20=%20" +
         "/FinanceStatusService.getFinanceStatusForjs?symbol=" + code + "&financeStatus=zero"
}

func GetUrlPrice(code string) string {
	return "http://hq.sinajs.cn/list=" + FullCode(code)
}

func httpGet(url string) (content string, err error) {
    resp, err := http.Get(url)
    if err != nil {
        fmt.Println("http get error", url)
        // handle error
        return "", err
    }
    if resp.StatusCode != 200 {
        fmt.Println("status code error", resp.StatusCode, url)
        // handle error
        return "", errors.New("bad_rsp_code")
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        fmt.Println("read body error", url)
        return "", err
    }
    // fmt.Println("code=", resp.StatusCode , "body_len=", len(string(body)))
    return string(body), nil
}

type Basic struct {
	  code string
	  name string
	  shConnect int
	  creditRating int
}

func LoadStockBasics() (stockBasics []*Basic, err error) {
    db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/magic_formula")
    defer db.Close()

    if err != nil {
        fmt.Printf("LoadStockBasics sql.Open err %s\r\n", err.Error())
        return nil, err
    }

    sql := `SELECT stock_code,stock_name,sh_connect, credit_rating FROM hk_stock_basic`

    rows, err := db.Query(sql)
    if err != nil {
        fmt.Printf("LoadStockBasics err %s\r\n", err.Error())
        return nil, err
    }
    for rows.Next() {
        var basic = Basic{}
        err = rows.Scan(&basic.code, &basic.name, &basic.shConnect, &basic.creditRating)
        if err != nil {
            fmt.Printf("LoadStockBasics err %s\r\n", err.Error())
            return nil, err
        }
        stockBasics = append(stockBasics, &basic)
    }

    return stockBasics, nil
}

func main() {
	  rand.Seed(int64(time.Now().Nanosecond()))
    basics, err := LoadStockBasics()
    if err != nil {
        fmt.Printf("LoadStockBasics err %v", err)
        return
    }
    fmt.Printf("total stock basics %d\r\n", len(basics))

    for i, basic := range basics {
        CrawlF10(basic.code)
        fmt.Println(i)
        // break
        interval := 10 + rand.Intn(50)
        time.Sleep(time.Duration(interval) * time.Millisecond)
    }
}

