#pragma once

#include <eosio/eosio.hpp>
#include <eosio/asset.hpp>
#include <eosio/system.hpp>


using namespace eosio;
using namespace std;

namespace mixin {

class [[eosio::contract("mtg.xin")]] mvmcontract : public eosio::contract {
public:
    using contract::contract;

    struct [[eosio::table]] tx_log {
        uint64_t id;
        uint64_t nonce;
        name contract;
        uint128_t process;
        uint128_t asset;
        std::vector<uint128_t> members;
        int32_t threshold;
        uint128_t amount;
        std::vector<uint8_t> extra;
        uint64_t timestamp;
        uint64_t primary_key()const { return id; }
        uint64_t get_contract_tx_request_nonce() const { return nonce; }
    };

    struct [[eosio::table]] counter {
        uint64_t id;
        uint64_t count;
        uint64_t primary_key()const { return id; }
    };

    struct [[eosio::table]] process {
        name contract;
        uint128_t process;
        uint64_t primary_key()const { return contract.value; }
    };

    typedef eosio::multi_index< "logs"_n, tx_log,
        eosio::indexed_by< "bytxnonce"_n, eosio::const_mem_fun<tx_log, uint64_t, &tx_log::get_contract_tx_request_nonce>>
    > tx_log_table;

    typedef eosio::multi_index<"counters"_n, counter> counter_table;
    typedef eosio::multi_index<"processes"_n, process> process_table;

    [[eosio::action]]
    void sayhello();

    [[eosio::action]]
    void addprocess(name contract, uint128_t process);

    [[eosio::action]]
    void txrequest(uint64_t nonce, name contract, uint128_t process, uint128_t asset,
                                vector<uint128_t> members, int32_t threshold, uint128_t amount,
                                vector<uint8_t> extra);

    [[eosio::action]]
    void ontxlog(ignore<tx_log>& log);

    uint64_t get_next_index(uint64_t key);
    uint64_t get_next_seq();

};


}
