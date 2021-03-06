# sql="select stock_code, stock_name, ROE from fiscal_year_2014 WHERE fiscal_year='2014' AND stock_name != '-' order by ROE DESC limit 50;"

# 根据2015年的配股情况，修正fiscal_year_2014表

while read line; do
	  code=$(echo $line | awk '{print $1}')
	  scale=$(echo $line | awk '{print $2}')
    # sql="UPDATE fiscal_year_2014 SET earning_per_share=earning_per_share/$scale,book_value_per_share=book_value_per_share/$scale,fixed=1 WHERE stock_code='$code' AND fixed=0;"
    sql="UPDATE stock_f10 SET share_split=$scale WHERE stock_code='$code';"
    echo $sql 
done < share_split_2015.conf | tee .share_split_2015.sql

cat .share_split_2015.sql | mysql -uroot magic_formula

