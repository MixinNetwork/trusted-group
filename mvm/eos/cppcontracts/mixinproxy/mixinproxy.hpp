#pragma once

#include <eosio/eosio.hpp>
#include <eosio/asset.hpp>
#include <eosio/system.hpp>
#include <eosio/crypto.hpp>
#include <eosio/singleton.hpp>

using namespace eosio;
using namespace std;

namespace mixin {
    typedef uint16_t weight_type;

    struct signup_public_key {
        uint8_t        type;
        array<unsigned char,33> data;
    };

    struct permission_level_weight {
        permission_level permission;
        weight_type weight;
    };

    struct key_weight {
        signup_public_key key;
        weight_type weight;
    };

    struct wait_weight {
        uint32_t wait_sec;
        weight_type weight;
    };

    struct authority {
        uint32_t threshold;
        vector<key_weight> keys;
        vector<permission_level_weight> accounts;
        vector<wait_weight> waits;
    };

    struct newaccount {
        name creator;
        name name;
        authority owner;
        authority active;
    };

    struct updateauth {
        name account;
        name permission;
        name parent;
        authority auth;
    };

class [[eosio::contract("mixinproxy")]] mixinproxy : public eosio::contract {
public:
    using contract::contract;

    struct tx_event {
        uint64_t nonce;
        uint128_t process;
        uint128_t asset;
        std::vector<uint128_t> members;
        int32_t threshold;
        uint128_t amount;
        std::vector<char> extra;
        uint64_t timestamp;
        vector<signature> signatures;
    };

    struct tx_request {
        uint64_t nonce;
        name contract;
        uint128_t process;
        uint128_t asset;
        std::vector<uint128_t> members;
        int32_t threshold;
        uint128_t amount;
        std::vector<uint8_t> extra;
        uint64_t timestamp;
    };

    struct [[eosio::table("counters")]] counter {
        uint64_t id;
        uint64_t count;
        uint64_t primary_key()const { return id; }
    };

    using counter_table = eosio::multi_index<"counters"_n, counter>;

    struct [[eosio::table("errorevents")]] error_tx_event {
        tx_event event;
        string reason;
        vector<char> origin_extra;
        uint64_t primary_key() const { return event.nonce; }
    };
    using error_tx_event_table = eosio::multi_index<"errorevents"_n, error_tx_event>;

    struct [[eosio::table("pendingevts")]] pending_event {
        tx_event event;
        name account;
        checksum256 hash;
        uint64_t primary_key() const { return event.nonce; }
        uint64_t get_account() const { return account.value; }
        checksum256 get_hash() const { return hash; }
    };

    using pending_event_table_type = eosio::multi_index<
            "pendingevts"_n,
            pending_event,
            eosio::indexed_by<"byaccount"_n,
                         eosio::const_mem_fun<pending_event, uint64_t, &pending_event::get_account>>,
            eosio::indexed_by<"byhash"_n,
                         eosio::const_mem_fun<pending_event, checksum256, &pending_event::get_hash>>
    >;

    struct [[eosio::table("submittedevs")]] submitted_event {
        uint64_t nonce;
        uint64_t primary_key() const { return nonce; }
    };

    using submitted_event_table_type = eosio::multi_index<"submittedevs"_n, submitted_event>;

    struct [[eosio::table("accountcache")]] account_cache {
        uint64_t id;
        name account;
    };
    using account_cache_table_type = eosio::singleton< "accountcache"_n, account_cache >;

    struct [[eosio::table("bindaccounts")]] mixin_account {
        name eos_account;
        uint128_t client_id;
        uint64_t primary_key() const { return eos_account.value; }
        uint128_t get_client_id() const { return client_id; }
    };

    using mixin_account_table_type = eosio::multi_index<
            "bindaccounts"_n,
            mixin_account,
            eosio::indexed_by<"byclientid"_n,
                         eosio::const_mem_fun<mixin_account, uint128_t, &mixin_account::get_client_id>>
    >;

    struct [[eosio::table]] mixin_asset {
        symbol symbol;
        uint128_t asset_id;
        uint64_t primary_key() const { return symbol.code().raw(); }
        uint128_t get_asset_id() const { return asset_id; }
    };

    using mixin_asset_table_type = eosio::multi_index<
            "mixinassets"_n,
            mixin_asset,
            eosio::indexed_by<"byassetid"_n,
                         eosio::const_mem_fun<mixin_asset, uint128_t, &mixin_asset::get_asset_id>>
    >;

    struct [[eosio::table]] transfer_fee {
        asset fee;
        uint64_t primary_key() const { return fee.symbol.code().raw(); }
    };
    using transfer_fee_table_type = eosio::multi_index<"transferfees"_n, transfer_fee>;

    struct [[eosio::table]] total_fee {
        asset total;
        uint64_t primary_key() const { return total.symbol.code().raw(); }
    };
    using total_fee_table_type = eosio::multi_index<"totalfees"_n, total_fee>;

    struct [[eosio::table("createaccfee")]] create_account_fee {
        asset fee;
    };
    // using create_account_fee_singleton_type = eosio::singleton<"createaccfee"_n, create_account_fee>;

    typedef eosio::singleton< "createaccfee"_n, create_account_fee >   create_account_fee_singleton_type;

    struct [[eosio::table]] process {
        name contract;
        uint128_t process;
        uint64_t primary_key() const { return contract.value; }
    };
    using process_table_type = eosio::multi_index<"processes"_n, process>;

    struct [[eosio::table]] signer {
        name account;
        public_key public_key;
        uint64_t primary_key() const { return account.value; }
        EOSLIB_SERIALIZE( signer, (account)(public_key))
    };
    using signer_table_type = eosio::multi_index<"signers"_n, signer>;

    struct operation {
        uint16_t purpose;
        uint128_t process;
        vector<char> platform;
        vector<char> address;
        vector<char> extra;
    };

    [[eosio::action("onevent")]]
    void on_event(ignore<tx_event>& event, ignore<vector<char>>& origin_extra);

    [[eosio::action("onerrorevent")]]
    void on_error_event(ignore<tx_event>& event, ignore<string>& reason, ignore<vector<char>>& origin_extra);

    [[eosio::action("initialize")]]
    void initialize();

    [[eosio::action("addasset")]]
    void add_mixin_asset(uint128_t asset_id, symbol symbol);

    [[eosio::action("removeasset")]]
    void remove_mixin_asset(symbol symbol);

    [[eosio::action("exec")]]
    void exec(name executor);

    [[eosio::action("execpending")]]
    void exec_pending_event_by_extra(name executor, uint64_t nonce, vector<char>& origin_extra);

    [[eosio::action("dowork")]]
    void doWork(name executor, uint64_t id);

    [[eosio::action("setfee")]]
    void set_transfer_fee(asset& fee);

    [[eosio::action("setaccfee")]]
    void set_create_account_fee(asset fee);

    [[eosio::action("ontransfer")]]
    void on_transfer(name from, name to, asset& quantity, string memo);

    [[eosio::action("error")]]
    void error(string err);

    //
    mixinproxy( name self, name first_receiver, datastream<const char*> ds );
    //
    uint64_t get_next_index(uint64_t key);
    //
    void check_and_inc_nonce(uint64_t old_nonce);
    //
    uint64_t get_next_tx_request_nonce();
    //
    operation decode_operation(const vector<char>& extra);
    //
    uint64_t get_nonce();
    //
    uint64_t inc_nonce();
    //
    void check_nonce(uint64_t event_nonce);
    //
    bool check_fee(tx_event& event);
    //
    bool check_expiration(const tx_event& event);
    //
    name check_account(const tx_event& event);
    //
    bool verify_signatures(const char* data, size_t size, vector<signature>& signatures);
    //
    name get_next_available_account();
    //
    void refund(const tx_event& event);
    //
    void create_new_account(name account);
    //
    symbol get_symbol(uint128_t asset_id);
    //
    bool issue_asset(const tx_event& event);
    //
    name get_account(uint128_t user_id);
    //
    void handle_event(const tx_event& event, const vector<char>& origin_extra);
    //
    action parse_action(vector<char> extra);
    //
    void handle_pending_event(const tx_event& event);
    //
    void show_error(string err);
    //
    asset get_transfer_fee(symbol sym);
    //
    void add_fee(asset a);

    private:
        uint128_t process_id;
};


}
