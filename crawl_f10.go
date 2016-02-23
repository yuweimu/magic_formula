package main

import (
	"fmt"
	"bufio"
	"os"
	"strings"
	"strconv"
	"io/ioutil"
  iconv "github.com/djimenez/iconv-go"
	"github.com/opesun/goquery"
	_ "github.com/go-sql-driver/mysql"
	"database/sql"
	"log"
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


func CrawlPrice(code string) (name, price string) {
	body, err := httpGet(GetUrlPrice(code))
	if err != nil {
		log.Fatal(err)
	}
	utf8, _ := iconv.ConvertString(body, "gb2312", "utf-8")
	fields := strings.Split(utf8, ",")
	if len(fields) < 4 {
		return "-", "-"
  }
	name = fields[0]
	idx := strings.Index(name, "\"")
	name = name[idx+1 :]
	price = fields[3]
	return name, price
}

var testF10 = `
<table class="needScroll">
  <tr>
   <th class="tips-colname-Left"><span>每股指标</span></th>
   <th class="tips-fieldname-Right"><span>14-12-31</span></th>
   <th class="tips-fieldname-Right"><span>13-12-31</span></th>
   <th class="tips-fieldname-Right"><span>12-12-31</span></th>
   <th class="tips-fieldname-Right"><span>11-12-31</span></th>
   <th class="tips-fieldname-Right"><span>10-12-31</span></th>
   <th class="tips-fieldname-Right"><span>09-12-31</span></th>
   <th class="tips-fieldname-Right"><span>08-12-31</span></th>
   <th class="tips-fieldname-Right"><span>07-12-31</span></th>
  </tr>
</table>
`

func CalculatePbPeRoe(price string, profitList string, bookValueList string) (pb float64, pe float64, roeList []float64) {
  profits := strings.Split(profitList, ",")
  bookValues := strings.Split(bookValueList, ",")
  fPrice, _ := strconv.ParseFloat(price, 32)
  pb = 0.0
  pe = 0.0
  for i := 1; i < len(profits); i++ {
      profit, err := strconv.ParseFloat(profits[i], 32)
      if err != nil {
          return
      }
      bookValue, err := strconv.ParseFloat(bookValues[i], 32)
      if err != nil {
          return
      }
      if i == 1 {
          pe = fPrice / profit
          pb = fPrice / bookValue
      }
      roeList = append(roeList, profit * 100 / bookValue)
  }

  return
}

func WriteDB(stockCode, stockName, year string, profit, bookValue, roe, grossProfitRate, netProfitRate float64, raw_data string) error {
    db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/magic_formula")
    defer db.Close()
    if err != nil {
        fmt.Printf("WriteDB sql.Open err %s", err.Error())
        return err
    }

    statement, err := db.Prepare("INSERT INTO stock_f10 SET stock_code=?,stock_name=?," +
                                 "fiscal_year=?,earning_per_share=?,book_value_per_share=?," +
                                 "gross_profit_rate=?,net_profit_rate=?,ROE=?,raw_data=?")
    if err != nil {
        fmt.Printf("WriteDB db.Prepare err %s", err.Error())
        return err
    }

    res, err := statement.Exec(stockCode, stockName, year, profit, bookValue,
                               grossProfitRate, netProfitRate, roe, raw_data)
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
func PersistF10(stockCode, stockName string, profitList, bookValueList, roeList, grossProfitRateList, netProfitRateList []string, raw_data string) {
    for i := 0; i < len(YearList); i++ {
        hasMore := false
        year := YearList[i]
        profit := ParseF10Field(profitList, i, &hasMore)
        bookValue := ParseF10Field(bookValueList, i, &hasMore)
        roe := ParseF10Field(roeList, i, &hasMore)
        grossProfitRate := ParseF10Field(grossProfitRateList, i, &hasMore)
        netProfitRate := ParseF10Field(netProfitRateList, i, &hasMore)
        fmt.Printf("xxxx %.3f,%.3f,%.3f,%.3f,%.3f\r\n", profit, bookValue, roe, grossProfitRate, netProfitRate)
        if hasMore {
            WriteDB(stockCode, stockName, year, profit, bookValue, roe, grossProfitRate, netProfitRate, raw_data)
            fmt.Printf("hasMore %d\r\n", i)
        }
    }
}

func FormatTrLine(trList *goquery.Nodes, index int) (result []string) {
  trProfit := trList.Slice(index, index + 1)
  tdList := trProfit.Find("td")
  tdCount := len(tdList)

  for j := 1; j <= tdCount; j++ {
    td := tdList.Slice(j, j + 1)
    text := td.Text()
    if len(text) <= 0 {
      break
    }
    result = append(result, text)
  }

  return result
}

func CrawlF10(name, price, code string) {
  body, err := httpGet(GetUrlF10(code))
  if err != nil {
    log.Fatal(err)
  }
  dom, _ := goquery.ParseString(body)
  trList := dom.Find("tr")

  profitList := FormatTrLine(&trList, 1)
  fmt.Println("基本每股收益", profitList)
  bookValueList := FormatTrLine(&trList, 4)
  fmt.Println("每股净资产", bookValueList)
  roeList := FormatTrLine(&trList, 20)
  fmt.Println("每股净资产", roeList)
  grossProfitRateList := FormatTrLine(&trList, 23)
  fmt.Println("毛利率", grossProfitRateList)
  netProfitRateList := FormatTrLine(&trList, 24)
  fmt.Println("净利率", netProfitRateList)

  PersistF10(code, name, profitList, bookValueList, roeList, grossProfitRateList, netProfitRateList, body)
}

func FullCode(code string) string {
	  if code[0] == '0' {
	      return "sz" + code
    }
	  return "sh" + code
}
func EastMoneyMarketCode(code string) string {
	if code[0] == '0' {
		return "02"
  }
	return "01"
}

func GetUrlF10(code string) string {
	unixMilli := fmt.Sprintf("%d", int64(time.Now().UnixNano() / 1e6))
  return "http://f10.eastmoney.com/f10_v2/BackOffice.aspx?command=RptF10MainTarget&code=" +
         code + EastMoneyMarketCode(code) + "&num=9&code1=" + FullCode(code) + "&spstr=&n=1&timetip=" + unixMilli
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
    file, err := os.Open("./stock_codes.conf")
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

func main() {
	  rand.Seed(int64(time.Now().Nanosecond()))
    codes, err := LoadStockCodes()
    if err != nil {
        fmt.Printf("LoadStockCodes err %v", err)
        return
    }
    fmt.Printf("total stock codes %d\r\n", len(codes))

  //for i := 0; i < 5; i++ {
  //    code := codes[i * 500 + 384]
    for i := 0; i < len(codes); i++ {
        code := codes[i]
        name, price := CrawlPrice(code)
        fmt.Printf("%s %s %s\r\n", code, name, price)
        CrawlF10(name, price, code)
        fmt.Println()
        interval := 10 + rand.Intn(50)
        time.Sleep(time.Duration(interval) * time.Millisecond)
    }
}

