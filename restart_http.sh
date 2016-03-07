ulimit -c 100000000
export LD_LIBRARY_PATH=../lib:$LD_LIBRARY_PATH

server=magic_formula_http

if [ ! -f $server ]; then
  echo "$server not found"
  exit
fi

count=$(ps axu | grep "${server}$" | wc -l)
if [ $count -eq 0 ];then
  echo "Service $server not running!"
else
  pid=$(ps axu | grep "${server}$" | awk '{print $2}')
  tput setaf 1
  echo "To stop $server $pid."
  kill -HUP $pid
  sleep 1.5
  echo "To kill $server $pid."
  kill -9 $pid
  sleep 0.3
  tput sgr0
fi
log_postfix=$(date "+%Y%m%d-%H%M%S")
# cd ../sbin && nohup ./$server -f ../etc/${server}.conf >& ../log/${server}.out.${log_postfix} &
nohup ./$server 2>&1 | grep -v "\[DEBUG\]" &> ${server}.out.${log_postfix} &
sleep 0.3
ps axu | grep -e "${server}$"
