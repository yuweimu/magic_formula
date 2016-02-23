package main

import (
  "fmt"
  "bufio"
  "os"
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
//http://stocks.sina.cn/hk/?code=03333&vt=4
// 取实时价格
//http://hq.sinajs.cn/list=hk03333
//var hq_str_hk03333="EVERGRANDE,恒大地产,5.340,5.290,5.350,5.210,5.220,-0.070,-1.323,5.210,5.240,23675395,4479000,4.883,8.238,8.400,3.040,2016/02/23,11:49";

func CrawlPrice(code string) (name, price, tradingDate string) {
    body, err := httpGet(GetUrlPrice(code))
    if err != nil {
        log.Fatal(err)
    }
    utf8, _ := iconv.ConvertString(body, "gb2312", "utf-8")
    fields := strings.Split(utf8, ",")
    if len(fields) < 18 {
        return "", "", ""
    }
    name = strings.Replace(fields[1], " ", "", -1)
    price = fields[6]
    tradingDate = strings.Replace(fields[17], "/", "-", -1)

    return name, price, tradingDate
}

func FullCode(code string) string {
    return "hk" + code
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

func LoadStockCodes() (stockCodes []string, err error) {
    file, err := os.Open("./hk_stock_codes.conf")
    if err != nil {
        fmt.Printf("LoadStockCodes err %v", err)
        return stockCodes, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        // fmt.Println(scanner.Text())
        stockCodes = append(stockCodes, scanner.Text())
    }
    if err := scanner.Err(); err != nil {
        fmt.Printf("LoadBusinessNames err %v", err)
        return stockCodes, err
    }

    return stockCodes, nil
}

func ClearDB(tradingDate string) error {
    db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/magic_formula")
    defer db.Close()

    statement, err := db.Prepare("DELETE FROM hk_stock_price WHERE trading_date=?")
    if err != nil {
        fmt.Printf("WriteDB db.Prepare err %s", err.Error())
        return err
    }

    _, err = statement.Exec(tradingDate)
    if err != nil {
        fmt.Printf("WriteDB db.Exec err %s", err.Error())
        return err
    }

    return nil
}

func WriteDB(stockCode, stockName, tradingDate string, price float64) error {
    db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/magic_formula")
    defer db.Close()
    if err != nil {
        fmt.Printf("WriteDB sql.Open err %s", err.Error())
        return err
    }

    statement, err := db.Prepare("INSERT INTO hk_stock_price SET stock_code=?,stock_name=?," +
                                 "trading_date=?,price=?")
    if err != nil {
        fmt.Printf("WriteDB db.Prepare err %s", err.Error())
        return err
    }

    res, err := statement.Exec(stockCode, stockName, tradingDate, price)
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
    codes, err := LoadStockCodes()
    if err != nil {
        fmt.Printf("LoadStockCodes err %v", err)
        return
    }
    fmt.Printf("total stock codes %d\r\n", len(codes))

    dbClearFlag := 0
    //for i := 0; i < 5; i++ {
    //    code := codes[i * 500]
    for i := 0; i < len(codes); i++ {
        code := codes[i]
        name, price, tradingDate := CrawlPrice(code)

        if len(name) <= 2 {
           fmt.Printf("CrawlError code=%s\r\n", code)
           continue
        }
        fmt.Printf("CrawlOk code=%s %s %s %s\r\n", code, name, price, tradingDate)
        fPrice, err := strconv.ParseFloat(price, 32)
        if err != nil {
            fPrice = 0.0
        }

        if dbClearFlag == 0 {
             ClearDB(tradingDate)
             dbClearFlag = 1
             time.Sleep(1000 * time.Millisecond)
        }

        WriteDB(code, name, tradingDate, fPrice)
        interval := 100 + rand.Intn(100)
        time.Sleep(time.Duration(interval) * time.Millisecond)
    }
}

