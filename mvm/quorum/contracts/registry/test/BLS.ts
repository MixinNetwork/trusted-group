// We import Chai to use its asserting functions here.
import { ethers } from "hardhat";
import { expect, assert } from "chai";
import { TestBLS } from "../typechain-types/test";
import BN from "bn.js";
import { randomHex } from "./bls/utils";
const mcl = require('./bls/mcl');

// `describe` is a Mocha function that allows you to organize your tests. It's
// not actually needed, but having your tests organized makes debugging them
// easier. All Mocha functions are available in the global scope.

// `describe` receives the name of a section of your test suite, and a callback.
// The callback must define the tests of that section. This callback can't be
// an async function.
describe("BLS library", function () {
  it("verify single signature", async function() {
    let BLS = await ethers.getContractFactory("TestBLS");
    let sbls = await BLS.deploy();

    await mcl.init();
    const message = randomHex(12);
    const { pubkey, secret } = mcl.newKeyPair();
    const [ signature, M ] = mcl.sign(message, secret);
    let message_ser = mcl.g1ToHex(M);
    let pubkey_ser = mcl.g2ToHex(pubkey);
    let sig_ser = mcl.g1ToHex(signature);
    let res = await sbls.verifySingle(sig_ser, pubkey_ser, message_ser);
    assert.isTrue(res);
  });
});
