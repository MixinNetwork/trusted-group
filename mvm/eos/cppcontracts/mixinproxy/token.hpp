#pragma once

#include <eosio/asset.hpp>
#include <eosio/symbol.hpp>
#include <eosio/eosio.hpp>
#include <eosio/singleton.hpp>
#include <eosio/crypto.hpp>

#include <string>
using namespace std;
using namespace eosio;

struct token_transfer {
    name    from;
    name    to;
    asset   quantity;
    string  memo;
    EOSLIB_SERIALIZE( token_transfer, (from)(to)(quantity)(memo) )
};

struct token_create {
    name   issuer;
    asset  maximum_supply;
    EOSLIB_SERIALIZE( token_create, (issuer)(maximum_supply))
};

struct token_issue {
    name to;
    asset quantity;
    string memo;
    EOSLIB_SERIALIZE( token_issue, (to)(quantity)(memo))
};

struct token_retire {
    asset quantity;
    string memo;
    EOSLIB_SERIALIZE( token_retire, (quantity)(memo))
};

struct token_open {
    name owner;
    symbol symbol;
    name ram_payer;
    EOSLIB_SERIALIZE( token_open, (owner)(symbol)(ram_payer))
};

struct token_close {
    name owner;
    symbol symbol;
    EOSLIB_SERIALIZE( token_close, (owner)(symbol))
};

struct [[eosio::table]] account {
    asset    balance;
    uint64_t primary_key()const { return balance.symbol.code().raw(); }
};

struct [[eosio::table]] currency_stats {
    asset    supply;
    asset    max_supply;
    name     issuer;
    uint64_t primary_key()const { return supply.symbol.code().raw(); }
};

typedef eosio::multi_index< "accounts"_n, account > accounts;
typedef eosio::multi_index< "stat"_n, currency_stats > stats;
