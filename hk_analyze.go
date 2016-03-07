package main

import (
    "fmt"
    "os"
    "sort"
    "math"
    "strings"
    "html/template"
    _ "github.com/go-sql-driver/mysql"
    "database/sql"
)

func Market(code string) string {
      if code[0] == '0' {
          return "sz"
    }
      return "sh"
}
func FullCode(code string) string {
      if code[0] == '0' {
          return "sz" + code
    }
      return "sh" + code
}

type Detail struct {
    code string
    name string
    price float64
    earning float64
    bookValue float64
    shareSplit float64

    pb float64
    pe float64
    roe1 float64
    roeList string
}
func (detail Detail) RoeList() string {
	return strings.Replace(detail.roeList, ",", " ", -1)
}
func (detail Detail) Pb() float64 {
    if detail.shareSplit > 0.001 {
        return detail.shareSplit * detail.pb
  }
  return detail.pb
}
func (detail Detail) Pe() float64 {
    if detail.shareSplit > 0.01 {
        return detail.shareSplit * detail.pe
  }
  return detail.pe
}

func (detail Detail) RankColor() string {
    if detail.shareSplit > 0.01 {
        return "green"
  }
    return "red"
}

func (detail Detail) Score() float64 {
      roe1 := math.Max(detail.roe1, 0.0001)
      pb := detail.Pb()
      if pb < 0.001 {
          pb = 10000
    }

    return pb / (math.Pow(roe1/10, 1.7))
}


func DbLoadPrice(tradingDate string) (*map[string]float64, error) {
    db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/magic_formula")
    defer db.Close()

    if err != nil {
        fmt.Printf("DbLoadPrice sql.Open err %s\r\n", err.Error())
        return nil, err
    }
    sql := fmt.Sprintf("SELECT stock_code,price FROM hk_stock_price WHERE trading_date='%s'", tradingDate)

    priceMap := make(map[string]float64)
    rows, err := db.Query(sql)
    if err != nil {
        fmt.Printf("DbLoadPrice err %s\r\n", err.Error())
        return nil, err
    }
    for rows.Next() {
        var code string
        var price float64

        err = rows.Scan(&code, &price)
        if err != nil {
            fmt.Printf("DbLoadPrice err %s\r\n", err.Error())
            return nil, err
        }
        // fmt.Printf("DbLoadPrice code=%s,price=%.3f\r\n", code, price)
        priceMap[code] = price
    }

    return &priceMap, nil
}

func DbLoadDetail(creditOnly bool) (result []*Detail, err error) {
    db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/magic_formula")
    defer db.Close()

    if err != nil {
        fmt.Printf("DbLoadDetail sql.Open err %s\r\n", err.Error())
        return nil, err
    }

    sql := `SELECT b.stock_code,b.stock_name,f.earning_per_share,f.book_value_per_share,
                   f.share_split, f.ROE_sina, f.ROE_list_sina
            FROM hk_stock_basic AS b, hk_stock_f10 AS f
            WHERE f.stock_code=b.stock_code AND f.fiscal_year IN ('2014') `
    if creditOnly {
        sql += "AND b.credit_rating > 0 "
    }
    sql += `ORDER BY f.ROE DESC LIMIT 3000;`

    rows, err := db.Query(sql)
    if err != nil {
        fmt.Printf("DbLoadDetail err %s\r\n", err.Error())
        return nil, err
    }
    for rows.Next() {
        var detail = Detail{}

        err = rows.Scan(&detail.code, &detail.name, &detail.earning, &detail.bookValue,
                        &detail.shareSplit, &detail.roe1, &detail.roeList)
        if err != nil {
            fmt.Printf("DbLoadDetail err %s\r\n", err.Error())
            return nil, err
        }
        // fmt.Printf("DbLoadDetail code=%s\r\n", detail.code)
        result = append(result, &detail)
    }

    return result, nil
}

type TemplateData struct {
    TradingDate string // 字段名首字母必须大写
    PageType string
}
var gHtmlHead = `<!DOCTYPE html PUBLIC "-//WAPFORUM//DTD XHTML Mobile 1.0//EN" "http://www.wapforum.org/DTD/xhtml-mobile10.dtd">
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
<style type="text/css">
body{
    font-size:14px;
    line-height:32px;
    font-weight:bold;
    font-family:"Courier New", Verdana, Arial, Sans-serif;
}
@media only screen and (max-device-width: 480px) {
    body {
        -webkit-text-size-adjust:100%;
    }   
}
</style>
</head>
<body>
<a href="/mf/home">Home</a>
<a href="/mf/{{.TradingDate}}.html">全部A股</a>
<a href="/mf/{{.TradingDate}}-exbank.html">非银A股</a>
<a href="/mf/{{.TradingDate}}-hk-credit.html">港股通</a>
<a href="/mf/{{.TradingDate}}-hk.html">全部H股</a><br />
{{.TradingDate}} {{.PageType}}<br />
<hr />
`

type DetailList []*Detail
func (s DetailList) Len() int      { return len(s) }
func (s DetailList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type SortByScore struct{ DetailList }
func (s SortByScore) Less(i, j int) bool {
      if s.DetailList[i].Score() < 0.0 {
          return false
    }
      return s.DetailList[i].Score() < s.DetailList[j].Score()
}

func main() {
    tradingDate := "2015-02-22"
    if len(os.Args) > 1 {
        tradingDate = os.Args[1]
    }
	  creditOnly := false
	  if len(os.Args) > 2 {
        creditOnly = true
    }

    priceMap, err := DbLoadPrice(tradingDate)
    if err != nil {
          os.Exit(1)
    }

    if len(*priceMap) <= 0 {
        fmt.Printf("trading date %s has no price info\r\n", tradingDate)
        os.Exit(1)
    }

    detailList, err := DbLoadDetail(creditOnly)
    if err != nil {
          os.Exit(1)
    }
    if len(detailList) <= 0 {
        fmt.Printf("no detail info\r\n")
        os.Exit(1)
    }

    tpl := template.New("header template")
    tpl, _ = tpl.Parse(gHtmlHead)
    p := TemplateData{TradingDate: tradingDate}
    if creditOnly {
        p.PageType = "港股通"
    } else {
        p.PageType = "全部H股"
    }
    tpl.Execute(os.Stdout, p)

    validDetailList := DetailList{}

    for _, detail := range detailList {
        if price, ok := (*priceMap)[detail.code]; ok {
            detail.price = price
            detail.pb = price / detail.bookValue
            detail.pe = price / detail.earning
            detail.roe1 = detail.earning * 100 / detail.bookValue
            validDetailList = append(validDetailList, detail)
        }
    }
    sort.Sort(SortByScore{validDetailList})

    excludedCodes := map[string]int {
        "01218":1,
        "08079":1,
        "00253":1,
        "03777":1,
        "00873":1,
        "01038":1,
        "00483":1,
        "02618":1,
        "02211":1,
        "01129":1,
        "01439":1,
        "01273":1,
        "01232":1,
        "00207":1,
        "01201":1,
      }

    rank := 0
    for _, detail := range validDetailList {
         if _, ok := excludedCodes[detail.code]; ok {
             continue
         }
         if detail.price < 0.60 {
             continue
         }
         rank++
         fmt.Printf("<span style=\"color:%s\">%d</span> %4.3f <span style=\"color:#292\">pb=%4.2f</span> pe=%4.2f" +
                    " <span style=\"color:#922\">roe1=%4.2f</span> roe=%s " +
                    "<a target=\"_blank\" href=\"http://stock.finance.sina.com.cn/hkstock/quotes/%s.html\">%s</a> " +
                    "<a target=\"_blank\" href=\"http://stocks.sina.cn/hk/?code=%s&vt=4\">手机版</a><br /><br />\r\n",
                    detail.RankColor(), rank, detail.Score(), detail.Pb(), detail.Pe(),
                    detail.roe1, detail.RoeList(), detail.code, detail.name, detail.code)
         if detail.Score() > 1.1 {
             break
         }
    }
    fmt.Printf("</body>\r\n</html>\r\n")
}

