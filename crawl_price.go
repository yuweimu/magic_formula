package main

import (
	"fmt"
	"strings"
	"io/ioutil"
  iconv "github.com/djimenez/iconv-go"
	_ "github.com/go-sql-driver/mysql"
	"database/sql"
	"log"
	"strconv"
	"time"
	"errors"
	"net/http"
	"math/rand"
)

// 取基本面信息
// http://f10.eastmoney.com/f10_v2/BackOffice.aspx?command=RptF10MainTarget&code=00089802&num=9&code1=sz000898&spstr=&n=1&timetip=1455959494972
// http://f10.eastmoney.com/f10_v2/BackOffice.aspx?command=RptF10MainTarget&code=00000202&num=9&code1=sz000002&spstr=&n=1&timetip=1455962256529
// http://f10.eastmoney.com/f10_v2/BackOffice.aspx?command=RptF10MainTarget&code=60116601&num=9&code1=sh601166&spstr=&n=1&timetip=1455962179556
//取实时价格
//http://hq.sinajs.cn/list=sh601006

func CrawlPrice(code string) (name, price, tradingDate string) {
    body, err := httpGet(GetUrlPrice(code))
    if err != nil {
        log.Fatal(err)
    }
    utf8, _ := iconv.ConvertString(body, "gb2312", "utf-8")
    fields := strings.Split(utf8, ",")
    if len(fields) < 32 {
        return "", "", ""
    }
    name = fields[0]
    idx := strings.Index(name, "\"")
    name = name[idx+1 :]
    name = strings.Replace(name, " ", "", -1)

    price = fields[3]
    if price == "0.00" {
        price = fields[2]
    }

    tradingDate = fields[30]
    return name, price, tradingDate
}

func FullCode(code string) string {
	  if code[0] == '0' {
	      return "sz" + code
    }
	  return "sh" + code
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
	  industry string
}

func LoadStockBasics() (stockCodes []*Basic, err error) {
    db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/magic_formula")
    defer db.Close()

    if err != nil {
        fmt.Printf("LoadStockBasics sql.Open err %s\r\n", err.Error())
        return nil, err
    }

    sql := `SELECT stock_code,stock_name,industry FROM stock_basic`

    rows, err := db.Query(sql)
    if err != nil {
        fmt.Printf("LoadStockBasics err %s\r\n", err.Error())
        return nil, err
    }
    for rows.Next() {
        var basic = Basic{}
        err = rows.Scan(&basic.code, &basic.name, &basic.industry)
        if err != nil {
            fmt.Printf("LoadStockBasics err %s\r\n", err.Error())
            return nil, err
        }
        stockCodes = append(stockCodes, &basic)
    }

    return stockCodes, nil
}

func ClearDB(tradingDate string) error {
    db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/magic_formula")
    defer db.Close()

    statement, err := db.Prepare("DELETE FROM stock_price WHERE trading_date=?")
    if err != nil {
        fmt.Printf("ClearDB db.Prepare err %s", err.Error())
        return err
    }

    _, err = statement.Exec(tradingDate)
    if err != nil {
        fmt.Printf("ClearDB db.Exec err %s", err.Error())
        return err
    }

    return nil
}

func WriteDB(stockCode, tradingDate string, price float64) error {
    db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/magic_formula")
    defer db.Close()
    if err != nil {
        fmt.Printf("WriteDB sql.Open err %s", err.Error())
        return err
    }

    statement, err := db.Prepare("INSERT INTO stock_price SET stock_code=?," +
                                 "trading_date=?,price=?")
    if err != nil {
        fmt.Printf("WriteDB db.Prepare err %s", err.Error())
        return err
    }

    res, err := statement.Exec(stockCode, tradingDate, price)
    if err != nil {
        fmt.Printf("WriteDB db.Exec err %s", err.Error())
        return err
    }

    insertId, err := res.LastInsertId()
    if err != nil {
        fmt.Printf("WriteDB err %s", err.Error())
        return err
    }
    fmt.Printf("WriteDB ok, stockCode=%s,insertId=%v\r\n", stockCode, insertId)
    return nil
}

func main() {
	  rand.Seed(int64(time.Now().Nanosecond()))
    basics, err := LoadStockBasics()
    if err != nil {
        fmt.Printf("LoadStockBasics err %v", err)
        return
    }
    fmt.Printf("total stock basics %d\r\n", len(basics))

    dbClearFlag := 0

    for i, basic := range basics {
        // if i % 500 != 0 {
        if false && i % 500 != 0 {
            continue
        }
        name, price, tradingDate := CrawlPrice(basic.code)

        if len(name) <= 2 {
           fmt.Printf("CrawlError code=%s\r\n", basic.code)
           continue
        }
        fmt.Printf("CrawlOk %s %s %s %s\r\n", basic.code, name, price, tradingDate)
        fPrice, err := strconv.ParseFloat(price, 32)
        if err != nil {
            fPrice = 0.0
        }

        if dbClearFlag == 0 {
             ClearDB(tradingDate)
             dbClearFlag = 1
             time.Sleep(1000 * time.Millisecond)
        }

        WriteDB(basic.code, tradingDate, fPrice)
        interval := 100 + rand.Intn(100)
        time.Sleep(time.Duration(interval) * time.Millisecond)
    }
}

