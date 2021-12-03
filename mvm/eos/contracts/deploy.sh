pushd dappdemo
./build.sh || exit 1
popd

pushd dappdemo2
./build.sh || exit 1
popd

pushd mtg.xin
./build.sh || exit 1
popd

python3 deploy.py || exit 1
