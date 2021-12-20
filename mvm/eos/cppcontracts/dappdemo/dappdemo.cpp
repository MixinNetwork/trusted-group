#include "dappdemo.hpp"

const uint64_t KEY_NONCE            = 1;
const uint64_t KEY_TX_REQUEST_INDEX = 2;
const uint64_t KEY_FINISHED_REQUEST = 3;
const name MTG_CONTRACT = "mtgxinmtgxin"_n;
const name MTG_PUBLISHER = "mtgpublisher"_n;

const uint64_t MAX_AMOUNT = (1ULL << 62) - 1;

//uuid: 49b00892-6954-4826-aaec-371ca165558a
const uint128_t PROCESS_ID = (uint128_t(0x8a5565a11c37ecaa)<<64) | uint128_t(0x264854699208b049);

namespace mixin {

[[eosio::action]]
void dappdemo::onevent(tx_event& event) {
    require_auth(MTG_PUBLISHER);
    check(event.process == PROCESS_ID, "Invalid process id");
    check_and_inc_nonce(event.nonce);
    
    uint64_t tx_request_count = 1;
    for (uint64_t i = 0; i < tx_request_count; i++) {
        uint64_t id = get_next_tx_request_nonce();
        check(event.amount < MAX_AMOUNT, "Invalid amount");
        auto amount = event.amount / tx_request_count;
		print("+++++++set amount:", amount, "\n");
        action{
            permission_level{get_self(), "active"_n},
            MTG_CONTRACT,
            "txrequest"_n,
            std::make_tuple(id, get_self(), PROCESS_ID, event.asset, event.members, event.threshold, amount, event.extra)
        }.send();
    }
}

uint64_t dappdemo::get_next_index(uint64_t key) {
    counter_table counters(get_self(), get_self().value);
    auto it = counters.find(key);
    if (it != counters.end() ) {
        auto count = it->count;
        counters.modify(it, get_self(), [&](auto& a) {
            a.count += 1;
        });
        return count;
    } else {
        counters.emplace(get_self(), [&](auto& a) {
            a.id = key;
            a.count = 1;
        });
        return 0;
    }
}

void dappdemo::check_and_inc_nonce(uint64_t old_nonce) {
    uint64_t key = KEY_NONCE;
    counter_table counters(get_self(), get_self().value);
    auto it = counters.find(key);
    if (it != counters.end()) {
        print("++++check_and_inc_nonce:", it->count, " ", old_nonce, "\n");
        check(it->count == old_nonce, "Invalid nonce");
        counters.modify(it, get_self(), [&](auto& a) {
            a.count = old_nonce + 1;
        });
    } else {
        counters.emplace(get_self(), [&](auto& a) {
            a.id = key;
            a.count = old_nonce + 1;
        });
    }
}

uint64_t dappdemo::get_next_tx_request_nonce() {
    return get_next_index(KEY_TX_REQUEST_INDEX);
}

}
