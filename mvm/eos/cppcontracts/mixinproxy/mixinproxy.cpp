#include "mixinproxy.hpp"
#include <eosio/datastream.hpp>
#include <eosio/time.hpp>
#include "token.hpp"

const uint64_t KEY_NONCE            = 1;
const uint64_t KEY_TX_REQUEST_INDEX = 2;
const uint64_t KEY_FINISHED_REQUEST = 3;
const name MTG_CONTRACT = "mtgxinmtgxin"_n;
const name MTG_PUBLISHER = "mtgpublisher"_n;
const name MIXIN_WTOKENS = "mixinwtokens"_n;
const name SYSTEM_CONTRACT = "eosio"_n;
const name ACCOUNT_CREATOR = "mixincrossss"_n;
const name FIRST_USER_ACCOUNT = "aaaaaaaaamvm"_n;

const uint64_t MTG_WORK_EXPIRATION_SECONDS = 3 * 60;
const uint32_t RAM_BYTES = 3 * 1024;
const uint64_t MAX_AMOUNT = (1ULL << 62) - 1;

//uuid: 49b00892-6954-4826-aaec-371ca165558a
const uint128_t PROCESS_ID = (uint128_t(0x8a5565a11c37ecaa)<<64) | uint128_t(0x264854699208b049);

namespace mixin {

mixinproxy::mixinproxy( name self, name first_receiver, datastream<const char*> ds )
: contract(self, first_receiver, ds)
{
    process_table_type processes(MTG_CONTRACT, MTG_CONTRACT.value);
    auto it = processes.find(self.value);
    check(it != processes.end(), "process not found!");
    this->process_id = it->process;
}

[[eosio::action("initialize")]]
void mixinproxy::initialize() {
    account_cache_table_type cache(get_self(), get_self().value);
    check(!cache.exists(), "contract has already been initialized");
    account_cache record{1, FIRST_USER_ACCOUNT};
    cache.set(record, get_self());
    this->create_new_account(FIRST_USER_ACCOUNT);
}

[[eosio::action("onevent")]]
void mixinproxy::on_event(ignore<tx_event>& event, ignore<vector<char>>& origin_extra) {
    //1. check fee
    //2. check expiration
    //3. check amount
    //4. check account
    //5. issue asset
    //6. decode action in extra

    tx_event _event;
    vector<char> _origin_extra;
    _ds >> _event;
    auto pos = _ds.tellp();
    _ds >> _origin_extra;

    _ds.seekp(0);
    bool ret = this->verify_signatures(_ds.pos(), pos - 1 - 66 * _event.signatures.size(), _event.signatures);
    check(ret, "not enough signatures.");
    check(this->process_id == _event.process, "Invalid process id");
    this->check_nonce(_event.nonce);
    if (!this->check_fee(_event)) {
        return;
    }

    if (this->check_expiration(_event)) {
        return;
    }

    check(_event.amount < MAX_AMOUNT, "amount exceed max limit!");

    if (_event.extra[0] == 1 && _origin_extra.size() == 0) {
        this->handle_pending_event(_event);
        return;
    }

    this->handle_event(_event, _origin_extra);
}

[[eosio::action("onerrorevent")]]
void mixinproxy::on_error_event(ignore<tx_event>& event, ignore<string>& reason, ignore<vector<char>>& origin_extra) {
    tx_event _event;
    string _reason;
    vector<char> _origin_extra;

    _ds >> _event;
    auto pos = _ds.tellp();
    _ds >> _reason;
    _ds >> _origin_extra;

    _ds.seekp(0);
    bool ret = this->verify_signatures(_ds.pos(), pos - 1 - 66 * _event.signatures.size(), _event.signatures);
    check(ret, "not enough signatures.");
    check(this->process_id == _event.process, "Invalid process id");
    this->check_nonce(_event.nonce);

    if (!this->check_fee(_event)) {
        return;
    }

    if (this->check_expiration(_event)) {
        return;
    }

    if (_event.extra[0] == 1 && _origin_extra.size() == 0) {
        this->handle_pending_event(_event);
        return;
    }

    error_tx_event_table table(get_self(), get_self().value);
    table.emplace(get_self(), [&](auto& row){
        row.event = _event;
        row.reason = _reason;
        row.origin_extra = _origin_extra;
    });
}


[[eosio::action("exec")]]
void mixinproxy::exec(name executor) {
    require_auth(executor);
    {
        pending_event_table_type table(get_self(), get_self().value);
        auto it = table.lower_bound(0);
        if (it != table.end()) {
            if (this->check_expiration(it->event)) {
                table.erase(it);
                return;
            }
        }
    }

    {
        error_tx_event_table table(get_self(), get_self().value);
        auto it = table.lower_bound(0);
        check(it != table.end(), "error event not found!");
        this->handle_event(it->event, it->origin_extra);
        table.erase(it);
    }
}

[[eosio::action("execpending")]]
void mixinproxy::exec_pending_event_by_extra(name executor, uint64_t nonce, vector<char>& origin_extra) {
    require_auth(executor);
    check(origin_extra.size() > 0, "origin_extra should not be empty");
    pending_event_table_type table(get_self(), get_self().value);
    auto it = table.find(nonce);
    check(it != table.end(), "pending event not found");
    this->handle_event(it->event, origin_extra);
    table.erase(it);
}

[[eosio::action("addasset")]]
void mixinproxy::add_mixin_asset(uint128_t asset_id, symbol symbol) {
    require_auth(get_self());
    mixin_asset_table_type table(get_self(), get_self().value);
    table.emplace(get_self(), [&](auto& row){
        row.symbol = symbol;
        row.asset_id = asset_id;
    });
}

//TODO:
[[eosio::action("removeasset")]]
void mixinproxy::remove_mixin_asset(symbol symbol) {
    require_auth(get_self());
    mixin_asset_table_type table(get_self(), get_self().value);
    auto it = table.find(symbol.code().raw());
    if (it == table.end()) {
        return;
    }
    table.erase(it);
}

[[eosio::action("dowork")]]
void mixinproxy::doWork(name executor, uint64_t id) {

}

[[eosio::action("setfee")]]
void mixinproxy::set_transfer_fee(asset& fee) {
    require_auth(get_self());
    mixin_asset_table_type mixin_assets(get_self(), get_self().value);
    auto it = mixin_assets.find(fee.symbol.code().raw());
    check(it != mixin_assets.end(), "asset not found!");

    transfer_fee_table_type table(get_self(), get_self().value);
    auto it2 = table.find(fee.symbol.code().raw());
    if (it2 == table.end()) {
        table.emplace(get_self(), [&](auto& row){
            row.fee = fee;
        });
    } else {
        table.modify(it2, get_self(), [&](auto& row) {
            row.fee = fee;
        });
    }
}

//TODO:
[[eosio::action("setaccfee")]]
void mixinproxy::set_create_account_fee(asset fee) {
}

[[eosio::action("ontransfer")]]
void mixinproxy::on_transfer(name from, name to, asset& quantity, string memo) {

}

[[eosio::action("error")]]
void mixinproxy::error(string err) {

}

uint64_t mixinproxy::get_next_index(uint64_t key) {
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

uint64_t mixinproxy::inc_nonce() {
    uint64_t key = KEY_NONCE;
    counter_table counters(get_self(), get_self().value);
    auto it = counters.find(key);
    if (it != counters.end()) {
        counters.modify(it, get_self(), [&](auto& row) {
            row.count += 1;
        });
        return it->count + 1;
    } else {
        counters.emplace(get_self(), [&](auto& row) {
            row.id = key;
            row.count = 1;
        });
        return 1;
    }
}

uint64_t mixinproxy::get_nonce() {
    uint64_t key = KEY_NONCE;
    counter_table counters(get_self(), get_self().value);
    auto it = counters.find(key);
    if (it != counters.end()) {
        return it->count;
    } else {
        counters.emplace(get_self(), [&](auto& row) {
            row.id = key;
            row.count = 1;
        });
        return 1;
    }
}

void mixinproxy::check_nonce(uint64_t event_nonce) {
    uint64_t nonce = this->get_nonce();
    check(event_nonce >= nonce, "bad nonce");

    submitted_event_table_type db(get_self(), get_self().value);
    auto it = db.find(event_nonce);
    check(it == db.end(), "event already exists!");
    db.emplace(get_self(), [&](auto& row){
        row.nonce = event_nonce;
    });

    for (;;) {
        auto it = db.find(nonce);
        if (it == db.end()) {
            break;
        }
        db.erase(it);
        this->inc_nonce();
        nonce += 1;
    }

    for (;;) {
        auto it = db.lower_bound(0);
        if (it == db.end()) {
            break;
        }

        if (it->nonce > nonce) {
            break;
        }
        db.erase(it);
    }
}

uint64_t mixinproxy::get_next_tx_request_nonce() {
    return get_next_index(KEY_TX_REQUEST_INDEX);
}

uint16_t read_uint16(datastream<const char*>& ds) {
    uint8_t c1;
    uint8_t c2;
    ds >> c1 >> c2;
    return (uint16_t(c1) << 8) | c2;
}

mixinproxy::operation mixinproxy::decode_operation(const vector<char>& extra) {
    uint8_t c;
    mixinproxy::operation op;
    datastream<const char*> ds(extra.data(), extra.size());
    // ds >> c;
    // check(c == 0, "bad extra type");

    op.purpose = read_uint16(ds);
    ds >> op.process;

    uint16_t length = read_uint16(ds);
    op.platform.resize(length);
    ds.read(op.platform.data(), length);

    length = read_uint16(ds);
    op.address.resize(length);
    ds.read(op.address.data(), length);

    length = read_uint16(ds);
    op.extra.resize(length);
    ds.read(op.extra.data(), length);
    return op;
}

bool mixinproxy::check_fee(tx_event& event) {
    auto sym = this->get_symbol(event.asset);
    auto fee = this->get_transfer_fee(sym);
    if (fee.amount == 0) {
        return true;
    }

    if (event.amount < uint128_t(fee.amount)) {
        return false;
    }

    event.amount -= uint128_t(fee.amount);
    this->add_fee(fee);
    return true;
}

bool mixinproxy::check_expiration(const tx_event& event) {
    if (event.timestamp / 1000000000 + MTG_WORK_EXPIRATION_SECONDS >= current_time_point().elapsed.to_seconds()) {
        return false;
    }
    this->refund(event);
    return true;
}

name mixinproxy::check_account(const tx_event& event) {
    name account = this->get_account(event.members[0]);
    if (account != name()) {
        return account;
    }

    auto sym = this->get_symbol(event.asset);
    if (sym != symbol("MEOS", 8)) {
        return name();
    }

    create_account_fee_singleton_type fee_table(get_self(), get_self().value);
    create_account_fee fee{asset(0, symbol("MEOS", 8))};
    fee = fee_table.get_or_default();
    if (event.amount < fee.fee.amount) {
        return name();
    }
    account = this->get_next_available_account();

    mixin_account_table_type mixin_accounts(get_self(), get_self().value);
    mixin_accounts.emplace(get_self(), [&](auto& row) {
        row.client_id = event.members[0];
        row.eos_account = account;
    });
    //return empty account in case of creating new account 
    return name{};
}

const char * alphabet = "abcdefghijklmnopqrstuvwxyz12345";
name mixinproxy::get_next_available_account() {
    account_cache_table_type cache(get_self(), get_self().value);
    auto record = cache.get();
    record.id += 1;
    vector<char> account = {'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'm', 'v', 'm'};
    auto id = record.id;
    int index = 8;
    while(id > 0) {
        char c = alphabet[int(id % 31)];
        id /= 31;
        account[index] = c;
        index -= 1;
    }
    auto current_account = record.account;
    record.account = name(string(account.data(), account.size()));
    cache.set(record, get_self());
    this->create_new_account(record.account);
    return current_account;
}

bool mixinproxy::verify_signatures(const char* data, size_t size, vector<signature>& signatures) {
    signer_table_type signers(MTG_CONTRACT, MTG_CONTRACT.value);
    vector<public_key> public_keys;
    auto it = signers.begin();
    for (;;) {
        if (it == signers.end()) {
            break;
        }
        public_keys.emplace_back(it->public_key);
        it++;
    }

    int valid_signatures = 0;
    int threshold = public_keys.size() * 2 / 3 + 1;
    auto hash = sha256(data, size);
    for (auto& sig: signatures) {
        auto pub_key = recover_key(hash, sig);
        auto it = std::find(public_keys.begin(), public_keys.end(), pub_key);
        if (it != public_keys.end()) {
            valid_signatures += 1;
            if (valid_signatures >= threshold) {
                return true;
            }
        }
    }
    return false;
}

void mixinproxy::create_new_account(name new_account_name) {
   authority owner = authority{
      .threshold = 1,
      .keys = {},
      .accounts = {
         {
            .permission = {get_self(), "owner"_n},
            .weight = 1,
         }
      },
      .waits = {}
   };

   authority active = authority{
      .threshold = 1,
      .keys = {},
      .accounts = {
         {
            .permission = {get_self(), "eosio.code"_n},
            .weight = 1,
         },
      },
      .waits = {}
   };

   newaccount new_account = newaccount{
      .creator = ACCOUNT_CREATOR,
      .name = new_account_name,
      .owner = owner,
      .active = active
   };

   action(
         permission_level{ _self, "active"_n },
         SYSTEM_CONTRACT,
         "newaccount"_n,
         new_account
   ).send();

   action(
         permission_level{ _self, "active"_n},
         SYSTEM_CONTRACT,
         "buyrambytes"_n,
         make_tuple(_self, new_account_name, RAM_BYTES)
   ).send();
}

void mixinproxy::refund(const tx_event& _event) {
    uint64_t tx_request_count = 1;
    for (uint64_t i = 0; i < tx_request_count; i++) {
        uint64_t id = get_next_tx_request_nonce();
        check(_event.amount < MAX_AMOUNT, "Invalid amount");
        action{
            permission_level{get_self(), "active"_n},
            MTG_CONTRACT,
            "txrequest"_n,
            std::make_tuple(id, get_self(), this->process_id, _event.asset, _event.members, _event.threshold, _event.amount, _event.extra)
        }.send();
    }
}

symbol mixinproxy::get_symbol(uint128_t asset_id) {
    mixin_asset_table_type table(get_self(), get_self().value);
    auto idx = table.get_index<"byassetid"_n>();
    auto it = idx.find(asset_id);
    if (it == idx.end()) {
        return symbol();
    }
    return it->symbol;
}

void mixinproxy::handle_event(const tx_event& event, const vector<char>& origin_extra) {
    if (this->check_expiration(event)) {
        return;
    }

    name from_account = this->check_account(event);
    if (from_account == name()) {
        return;
    }

    action a;
    if (event.extra[0] == 0) {
        a = this->parse_action(event.extra);
    } else if (event.extra[0] == 1) {
        if (event.extra.size() <= 1 + 32) {
            return;
        }
        check (origin_extra.size() != 0, "origin_extra should not be empty!!");
        checksum256 origin_hash{};
        {
            datastream<char*> ds((char *)event.extra.data() + 1, 32);
            ds >> origin_hash;
        }
        auto hash = sha256(origin_extra.data(), origin_extra.size());
        check(origin_hash == hash, "extra hash mismatch");
        if (origin_hash != hash) {
            this->show_error("extra hash mismatch");
            return;
        }
        auto op = decode_operation(origin_extra);
        a = this->parse_action(op.extra);
    }

    this->issue_asset(event);

    if (a.account == name()) {
        return;
    }
    a.authorization.emplace_back(permission_level(from_account, "active"_n));
    a.send();
}

bool mixinproxy::issue_asset(const tx_event& event) {
    uint128_t asset_id = event.asset;
    uint128_t amount = event.amount;

    auto sym = this->get_symbol(asset_id);
    if (sym.raw() == 0) {
        return false;
    }

    auto account = this->get_account(event.members[0]);
    stats stat_table(MIXIN_WTOKENS, sym.code().raw());
    auto it = stat_table.find(sym.code().raw());
    if (it == stat_table.end()) {
        action{
            permission_level{ MIXIN_WTOKENS, "active"_n },
            MIXIN_WTOKENS,
            "create"_n,
            std::make_tuple(get_self(), asset(MAX_AMOUNT, sym)),
        }.send();
    }

    asset a(int64_t(amount), sym);

    action{
        permission_level{ get_self(), "active"_n },
        MIXIN_WTOKENS,
        "issue"_n,
        std::make_tuple(get_self(), a, string("issue")),
    }.send();

    action{
        permission_level{ get_self(), "active"_n },
        MIXIN_WTOKENS,
        "transfer"_n,
        std::make_tuple(get_self(), account, a, string("transfer")),
    }.send();
    return true;
}

name mixinproxy::get_account(uint128_t user_id) {
    mixin_account_table_type mixin_accounts(get_self(), get_self().value);
    auto idx = mixin_accounts.get_index<"byclientid"_n>();
    auto it = idx.find(user_id);
    if (it != idx.end()) {
        return it->eos_account;
    }
    return name();
}

action mixinproxy::parse_action(vector<char> extra) {
    action a;
    uint8_t c;
    datastream<char*> ds(extra.data(), extra.size());

    ds >> c;
    if (c != 0) {
        return a;
    }

    ds >> a.account;
    ds >> a.name;
    a.data.resize(ds.remaining());
    memcpy(a.data.data(), ds.pos(), ds.remaining());
    return a;
}

//TODO:
void mixinproxy::handle_pending_event(const tx_event& event) {
    name from_account = this->check_account(event);
    if (from_account == name()) {
        return;
    }

    pending_event_table_type table(get_self(), get_self().value);
    auto it = table.find(event.nonce);
    check(it == table.end(), "pending event already exists!");
    checksum256 hash;
    memcpy(hash.data(), event.extra.data() + 1, 32);
    table.emplace(get_self(), [&](auto& row){
        row.event = event;
        row.account = from_account;
        row.hash = hash;
    });
}

//TODO:
void mixinproxy::show_error(string err) {

}

asset mixinproxy::get_transfer_fee(symbol sym) {
    transfer_fee_table_type table(get_self(), get_self().value);
    auto it = table.find(sym.code().raw());
    if (it == table.end()) {
        return asset();
    }
    return it->fee;
}

void mixinproxy::add_fee(asset a) {
    total_fee_table_type table(get_self(), get_self().value);
    auto it = table.find(a.symbol.code().raw());
    if (it == table.end()){
        table.emplace(get_self(), [&](auto& row) {
            row.total = a; 
        });
    } else {
        table.modify(it, get_self(), [&](auto& row) {
            row.total += a;
        });
    }
}

// addFee(asset: Asset): void {
//     let db = TotalFee.new(this.receiver, this.receiver);
//     let it = db.find(asset.symbol.code())
//     if (it.isOk()) {
//         let record = db.get(it);
//         record.total += asset;
//         db.update(it, record, new Name());
//     } else {
//         db.store(new TotalFee(asset), this.receiver);
//     }
// }

}
