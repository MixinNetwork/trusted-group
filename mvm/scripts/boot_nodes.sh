python3 ../../mvm-configs/update.py || exit 1
pidof mvm | xargs kill
rm -r test
mkdir -p test
nohup ./mvm boot -c ../../mvm-configs/test1.toml -d ./test/data1  >> test/test1.log &
nohup ./mvm boot -c ../../mvm-configs/test2.toml -d ./test/data2  >> test/test2.log &
nohup ./mvm boot -c ../../mvm-configs/test3.toml -d ./test/data3  >> test/test3.log &
nohup ./mvm boot -c ../../mvm-configs/test4.toml -d ./test/data4  >> test/test4.log &
