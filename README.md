# magic_formula
1. 获取基本面信息, 写入DB
./crawl_f10
./fix_f10.sh #根据配股数据修正每股盈利、每股净资产

2. 获取价格信息, 写入DB
./crawl_price

3. 从1，2抓取结果中计算pb/pe/roe数据
./analyze


