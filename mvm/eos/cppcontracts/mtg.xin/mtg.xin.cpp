#include "mtg.xin.hpp"

constexpr uint64_t KEY_TX_REQUEST_SEQ = 1;

namespace mixin {

[[eosio::action]]
void mvmcontract::sayhello() {
    print("Hello, world!");
}

[[eosio::action]]
void mvmcontract::addprocess(name contract, uint128_t process) {
    require_auth(get_self());
    check(is_account(contract), "contract account does not exists!");
    process_table processes(get_self(), get_self().value);
    auto it = processes.find(contract.value);
    check(it == processes.end(), "process already exists!");
    processes.emplace(get_self(), [&](auto &row) {
        row.contract = contract;
        row.process = process;
    });
}

[[eosio::action]]
void mvmcontract::txrequest(uint64_t nonce, name contract, uint128_t process, uint128_t asset,
                            vector<uint128_t> members, int32_t threshold, uint128_t amount,
                            vector<uint8_t> extra) {
    require_auth(contract);
    process_table processes(get_self(), get_self().value);
    auto it = processes.find(contract.value);
    check(it != processes.end(), "process not found!");
    check(it->process == process, "invalid process!!");

    uint64_t seq = get_next_seq();

    action(
            permission_level{get_self(), "active"_n},
            get_self(),
            "ontxlog"_n,
            std::make_tuple(seq, nonce, contract, process, asset, members, threshold, amount, extra, uint64_t(current_time_point().elapsed.count()*1000))
    ).send();

}

[[eosio::action]]
void mvmcontract::ontxlog(ignore<tx_log>& log) {
    require_auth(get_self());
}

uint64_t mvmcontract::get_next_index(uint64_t key) {
    counter_table counters(get_self(), get_self().value);
    auto it = counters.find(key);
    if (it != counters.end()) {
        uint64_t index = it->count;
        counters.modify(it, get_self(), [&](auto &row) {
            row.count += 1;
        });
        return index;
    } else {
        counters.emplace(get_self(), [&](auto &row) {
            row.id = key;
            row.count = 1;
        });
        return 0;
    }
}

uint64_t mvmcontract::get_next_seq() {
    return get_next_index(KEY_TX_REQUEST_SEQ);
}

}
