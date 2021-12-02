#pragma once

#include <eosio/eosio.hpp>
#include <eosio/asset.hpp>
#include <eosio/system.hpp>


using namespace eosio;
using namespace std;

namespace mixin {

class [[eosio::contract("dappdemo")]] dappdemo : public eosio::contract {
public:
    using contract::contract;

    struct tx_event {
        uint64_t nonce;
        uint128_t process;
        uint128_t asset;
        std::vector<uint128_t> members;
        int32_t threshold;
        uint128_t amount;
        std::vector<uint8_t> extra;
        uint64_t timestamp;
        std::vector<uint8_t> signature;
    };

    struct tx_request {
        uint64_t nonce;
        uint128_t process;
        uint128_t asset;
        std::vector<uint128_t> members;
        int32_t threshold;
        uint128_t amount;
        std::vector<uint8_t> extra;
        uint64_t timestamp;
    };

    struct [[eosio::table]] counter {
        uint64_t id;
        uint64_t count;
        uint64_t primary_key()const { return id; }
    };

    typedef eosio::multi_index<"counters"_n, counter> counter_table;

    [[eosio::action]]
    void onevent(tx_event& event);

    uint64_t get_next_index(uint64_t key);
    void check_and_inc_nonce(uint64_t old_nonce);
    uint64_t get_next_tx_request_nonce();
};


}
