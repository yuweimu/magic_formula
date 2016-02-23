package main

import (
	"fmt"
	"bufio"
	"os"
	"strings"
	"strconv"
	"io/ioutil"
  iconv "github.com/djimenez/iconv-go"
	_ "github.com/go-sql-driver/mysql"
  "github.com/PuerkitoBio/goquery"
	"database/sql"
	"log"
	"time"
	"errors"
	"net/http"
	"math/rand"
)

//取基本面信息
//http://quotes.money.163.com/hkstock/cwsj_03333.html
//取实时价格
//http://hq.sinajs.cn/list=hk03333
//var hq_str_hk03333="EVERGRANDE,恒大地产,5.340,5.290,5.350,5.210,5.220,-0.070,-1.323,5.210,5.240,23675395,4479000,4.883,8.238,8.400,3.040,2016/02/23,11:49";

func CrawlPrice(code string) (name, price string) {
    body, err := httpGet(GetUrlPrice(code))
    if err != nil {
        log.Fatal(err)
    }
    utf8, _ := iconv.ConvertString(body, "gb2312", "utf-8")
    fields := strings.Split(utf8, ",")
    if len(fields) < 18 {
        return "-", "-"
    }
    name = strings.Replace(fields[1], " ", "", -1)
    price = fields[6]

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

func ExtractTrLine(trSel *goquery.Selection) float64 {
	  value := 0.0
    trSel.Find("td").Each(func(i int, tdSel *goquery.Selection) {
        if i == 1 {
            str := strings.Replace(tdSel.Text(), "%", "", -1)
            value, _ = strconv.ParseFloat(str, 32)
        }
    })
    return value
}

//http://quotes.money.163.com/hkstock/cwsj_03333.html
//<table width="20%" class="mod-table2 column" cellpadding="0" cellspacing="0">
//  <tr> <td class="th">报表日期</td> </tr>
//  <tr> <td class="th">基本每股收益</td> </tr>
//  <tr> <td class="th">摊薄每股收益</td> </tr>
//  <tr> <td class="th">毛利率</td> </tr>
//  <tr> <td class="th">贷款回报率</td> </tr>
//  <tr> <td class="th">总资产收益率</td> </tr>
//  <tr> <td class="th">净资产收益率</td> </tr>
//  <tr> <td class="th">流动比率</td> </tr>
//  <tr> <td class="th">速动比率</td> </tr>
//  <tr> <td class="th">资本充足率</td> </tr>
//  <tr> <td class="th">资产周转率</td> </tr>
//  <tr> <td class="th">存贷比</td> </tr>
//  <tr> <td class="th">存货周转率</td> </tr>
//  <tr> <td class="th">管理费用比率</td> </tr>
//  <tr> <td class="th">财务费用比率</td> </tr>
//  <tr> <td class="th">销售现金比率</td> </tr>
//</table>
//<table class="mod-table2 thWidth205" cellpadding="0" cellspacing="0" >
//  <tbody id='cwzb'>
//    <tr> <td><div class="align-c">2015-06-30</div></td><td><div class="align-c">2014-12-31</div></td><td><div class="align-c">2014-06-30</div></td> </tr>
//    <tr> <td><div>0.79</div></td><td><div>1.07</div></td><td><div>0.60</div></td> </tr> // 基本每股收益 
//    <tr> <td><div>0.78</div></td><td><div>1.06</div></td><td><div>0.59</div></td> </tr>
//    <tr> <td><div>28.39%</div></td><td><div>28.53%</div></td><td><div>28.57%</div></td> </tr> // 毛利率 
//    <tr> <td><div>--</div></td><td><div>--</div></td><td><div>--</div></td> </tr>
//    <tr> <td><div>3.49%</div></td><td><div>2.66%</div></td><td><div>3.36%</div></td> </tr> // 总资产收益率 
//    <tr> <td><div>17.07%</div></td><td><div>12.12%</div></td><td><div>15.67%</div></td> </tr> // ROE
//    <tr> <td><div>1.32</div></td><td><div>1.43</div></td><td><div>1.44</div></td> </tr>
//    <tr> <td><div>0.43</div></td><td><div>0.51</div></td><td><div>0.53</div></td> </tr>
//    <tr> <td><div>--</div></td><td><div>--</div></td><td><div>--</div></td> </tr>
//    <tr> <td><div>0.14</div></td><td><div>0.23</div></td><td><div>0.15</div></td> </tr>
//    <tr> <td><div>--</div></td><td><div>--</div></td><td><div>--</div></td> </tr>
//    <tr> <td><div>0.54</div></td><td><div>0.45</div></td><td><div>0.58</div></td> </tr>
//    <tr> <td><div>33.77%</div></td><td><div>27.68%</div></td><td><div>33.39%</div></td> </tr>
//    <tr> <td><div>2.80%</div></td><td><div>3.33%</div></td><td><div>0.06%</div></td> </tr>
//    <tr> <td><div>0.95%</div></td><td><div>1.87%</div></td><td><div>0.99%</div></td> </tr>
//  </tbody>
//</table>
func CrawlF10(name, code string) {
    doc, err := goquery.NewDocument(GetUrlF10(code))
    if err != nil {
        log.Fatal(err)
    }
    profit := 0.0
    grossProfitRate := 0.0
    roa := 0.0
    roe := 0.0
    doc.Find("#cwzb tr").Each(func(i int, trSel *goquery.Selection) {
        if i == 1 {
            profit = ExtractTrLine(trSel)
        }
        if i == 3 {
            grossProfitRate = ExtractTrLine(trSel)
        }
        if i == 5 {
            roa = ExtractTrLine(trSel)
        }
        if i == 6 {
            roe = ExtractTrLine(trSel)
        }
    })


    fmt.Printf("profit=%.3f,grossProfitRate=%.3f,roa=%.3f,roe=%.3f", profit, grossProfitRate, roa, roe)
    return
  // PersistF10(code, name, profitList, bookValueList, roeList, grossProfitRateList, netProfitRateList, body)
}

func FullCode(code string) string {
	  return "hk" + code
}

func GetUrlF10(code string) string {
  return "http://quotes.money.163.com/hkstock/cwsj_" + code + ".html"
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
        CrawlF10(name, code)
        fmt.Println()
        interval := 10 + rand.Intn(50)
        time.Sleep(time.Duration(interval) * time.Millisecond)
    }
}

