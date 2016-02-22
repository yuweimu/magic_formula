week_day=$(date +%w)
today=$(date +%Y-%m-%d)

if [ $week_day -eq 0 -o $week_day -eq 6 ];then
	echo "$today : week_day $week_day is not a trading day."
	exit
fi

./crawl_price
./analyze > $today.html

if [ $? -eq 0 ];then
	mv -fv $today.html /data/nginx/html/mf/.
else
	rm -fv $today.html
fi
