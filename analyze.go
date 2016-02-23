package main

import (
	"fmt"
	"os"
	"sort"
	"math"
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
    earning float64
    bookValue float64
    share_split float64
    roe3 float64

    pb float64
    pe float64
    roe1 float64
}
func (detail Detail) Pb() float64 {
	if detail.share_split > 0.001 {
		return detail.share_split * detail.pb
  }
  return detail.pb
}
func (detail Detail) Pe() float64 {
	if detail.share_split > 0.01 {
		return detail.share_split * detail.pe
  }
  return detail.pe
}

func (detail Detail) RankColor() string {
	if detail.share_split > 0.01 {
		return "green"
  }
	return "red"
}

func (detail Detail) Score() float64 {
	  roe1 := math.Max(detail.roe1, 0.0001)
	  roe3 := math.Max(detail.roe3, 0.0001)
	  pb := detail.Pb()
	  if pb < 0.001 {
	      pb = 10000
    }

    return pb / (math.Pow(roe1/10, 0.8) * math.Pow(roe3/10, 0.9))
}


func DbLoadPrice(tradingDate string) (*map[string]float64, error) {
    db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/magic_formula")
    defer db.Close()

    if err != nil {
        fmt.Printf("DbLoadPrice sql.Open err %s\r\n", err.Error())
        return nil, err
    }
    sql := fmt.Sprintf("SELECT stock_code,price FROM stock_price WHERE trading_date='%s'", tradingDate)

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

func DbLoadDetail() (result []*Detail, err error) {
    db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/magic_formula")
    defer db.Close()

    if err != nil {
        fmt.Printf("DbLoadDetail sql.Open err %s\r\n", err.Error())
        return nil, err
    }

    sql := `SELECT stock_code,stock_name,earning_per_share,book_value_per_share,share_split,sum(ROE)/3 AS average_ROE from stock_f10
            WHERE fiscal_year IN ('2012', '2013', '2014') AND stock_name != '-' 
            GROUP BY stock_code ORDER BY average_ROE DESC 
            limit 1000;`

    rows, err := db.Query(sql)
    if err != nil {
        fmt.Printf("DbLoadDetail err %s\r\n", err.Error())
        return nil, err
    }
    for rows.Next() {
        var detail = Detail{}

        err = rows.Scan(&detail.code, &detail.name, &detail.earning, &detail.bookValue, &detail.share_split, &detail.roe3)
        if err != nil {
            fmt.Printf("DbLoadDetail err %s\r\n", err.Error())
            return nil, err
        }
        // fmt.Printf("DbLoadDetail code=%s\r\n", detail.code)
        result = append(result, &detail)
    }

    return result, nil
}

var gHtmlHead = `<!DOCTYPE html PUBLIC "-//WAPFORUM//DTD XHTML Mobile 1.0//EN" "http://www.wapforum.org/DTD/xhtml-mobile10.dtd">
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
<style type="text/css">
body{
    font-size:18px;
    line-height:42px;
    font-weight:bold;
    font-family:"Courier New", Verdana, Arial, Sans-serif;
}
@media only screen and (max-device-width: 480px) {
    body {
        -webkit-text-size-adjust:70%;
    }   
}
</style>
</head>
<body>

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
	  tradingDate := "2016-02-22"
	  if len(os.Args) > 1 {
        tradingDate = os.Args[1]
    }

	  priceMap, err := DbLoadPrice(tradingDate)
	  if err != nil {
	      os.Exit(1)
    }

    if len(*priceMap) <= 0 {
        fmt.Printf("trading date %s has no price info\r\n", tradingDate)
	      os.Exit(1)
    }

	  detailList, err := DbLoadDetail()
	  if err != nil {
	      os.Exit(1)
    }
    if len(detailList) <= 0 {
        fmt.Printf("no detail info\r\n")
	      os.Exit(1)
    }
    fmt.Printf("%s", gHtmlHead)
    validDetailList := DetailList{}

    for _, detail := range detailList {
        if price, ok := (*priceMap)[detail.code]; ok {
            detail.pb = price / detail.bookValue
            detail.pe = price / detail.earning
            detail.roe1 = detail.earning * 100 / detail.bookValue

            // fmt.Printf("%4.3f pb=%4.3f pe=%4.3f roe1=%4.3f roe3=%4.3f %s\r\n", score, pb, pe, roe1, detail.roe3, detail.code)
         // fmt.Printf("%4.3f <span style=\"color:#292\">pb=%4.3f</span> pe=%4.3f " +
         //            "<br /> &nbsp; &nbsp; <span style=\"color:#922\">roe1=%4.3f</span> roe3=%4.3f " + 
         //            "<a href=\"http://stocks.sina.cn/sh/?code=%s\">%s</a><br />\r\n",
         //            score, pb, pe, roe1, detail.roe3, FullCode(detail.code), detail.code)
            validDetailList = append(validDetailList, detail)
        }
    }
    sort.Sort(SortByScore{validDetailList})

    for rank, detail := range validDetailList {
       //fmt.Printf("%4.3f pb=%4.3f pe=%4.3f roe1=%4.3f roe3=%4.3f %s\r\n", detail.Score(),
       //           detail.pb, detail.pe, detail.roe1, detail.roe3, detail.code)
         fmt.Printf("<span style=\"color:%s\">%d</span> %4.3f <span style=\"color:#292\">pb=%4.3f</span> pe=%4.3f " +
                    " <span style=\"color:#922\">roe1=%4.3f</span> <br /> &nbsp; &nbsp; roe3=%4.3f " + 
                    //"<a href=\"http://stocks.sina.cn/sh/?code=%s\">%s</a><br />\r\n",
                    "<a href=\"http://stocks.sina.cn/sh/finance?vt=4&code=%s\">分红配股</a> " +
                    "<a href=\"http://finance.sina.com.cn/realstock/company/%s/nc.shtml\">%s</a><br /><br />\r\n",
                    detail.RankColor(), rank + 1, detail.Score(), detail.Pb(), detail.Pe(),
                    detail.roe1, detail.roe3, FullCode(detail.code), FullCode(detail.code), detail.name)
    }
    fmt.Printf("</body>\r\n</html>\r\n")
}

