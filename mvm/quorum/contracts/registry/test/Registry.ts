// We import Chai to use its asserting functions here.
import { ethers } from "hardhat";
import { expect, assert } from "chai";
import { Signer } from 'ethers';
import { Registry } from "../typechain-types";
import { TestBLS } from "../typechain-types/test";
import { randomHex } from "./bls/utils";
import BN from "bn.js";
const mcl = require('./bls/mcl');

// `describe` is a Mocha function that allows you to organize your tests. It's
// not actually needed, but having your tests organized makes debugging them
// easier. All Mocha functions are available in the global scope.

// `describe` receives the name of a section of your test suite, and a callback.
// The callback must define the tests of that section. This callback can't be
// an async function.
describe("Registry contract", function () {
  // Mocha has four functions that let you hook into the test runner's
  // lifecyle. These are: `before`, `beforeEach`, `after`, `afterEach`.

  // They're very useful to setup the environment for tests, and to clean it
  // up after they run.

  // A common pattern is to declare some variables, and assign them in the
  // `before` and `beforeEach` callbacks.

  let registry: Registry;
  let PID = '0xb45dcee023d74ad1b51ec681a257c13e';
  let GROUP: any;
  let SIGNER: any;
  let addr1: Signer;
  let addr2;
  let addrs;

  // `beforeEach` will run before each test, re-deploying the contract every
  // time. It receives a callback, which can be async.
  beforeEach(async function () {
    [addr1, addr2, ...addrs] = await ethers.getSigners();
    let Registry = await ethers.getContractFactory("Registry", {
      //libraries: {
      //BLS: sbls.address,
      //},
    });
    // To deploy our contract, we just have to call Token.deploy() and await
    // for it to be deployed(), which happens once its transaction has been
    // mined.
    await mcl.init();
    let {pubkey, secret} = mcl.newKeyPair();
    let pubkey_ser = mcl.g2ToUnifiedHex(pubkey);
    registry = await Registry.connect(addr1).deploy(pubkey_ser, PID);
    GROUP = pubkey;
    SIGNER = secret;
  });

  describe("Test mixin function", function () {
    it("Should have the same PID", async function () {
      expect(await registry.PID()).to.equal(PID);
    });

    it("Should have the same GROUP", async function () {
      for (let i = 0; i < 4; i++) {
        let sge = await registry.GROUP(i);
        let tge = mcl.g2ToBN(GROUP)[i];
        expect(sge.toString()).to.equal(tge.toString());
      }
    });

    it("Should fail with error NONCE", async function () {
      let raw = '0xb45dcee023d74ad1b51ec681a257c13e' + // PID
        '0000000000000006' + // NONCE
        'c6d0c7282624429b8e0dd9d19b6592fa' + // asset
        '000301e240000e00034254430007426974636f696e' + // amount + extra
        '16f90cc73f2b75f0' + // timestamp
        '0001fcb874914fa04c2fb387262b63cbc1120001' + // members
        '0040246f86caf3a5d195c471e82cc73fe10606c97d2f6a79b19920bdadc9699eb10222cae85bb19a4ff719540213b51379acf1e5baad1917a27a90be28239fec1616';
      await expect(registry.mixin(raw)).to.be.revertedWith("invalid nonce");
    });

    it("Should fail with error signature", async function () {
      let raw = '0xb45dcee023d74ad1b51ec681a257c13e' + // PID
        '0000000000000000' + // NONCE
        'c6d0c7282624429b8e0dd9d19b6592fa' + // asset
        '000301e240000e00034254430007426974636f696e' + // amount + extra
        '16f90cc73f2b75f0' + // timestamp
        '0001fcb874914fa04c2fb387262b63cbc1120001' + // members
        '0040246f86caf3a5d195c471e82cc73fe10606c97d2f6a79b19920bdadc9699eb10222cae85bb19a4ff719540213b51379acf1e5baad1917a27a90be28239fec1616';
      await expect(registry.mixin(raw)).to.be.revertedWith("invalid signature");
    });

    it("Should fail with error signature", async function () {
      let raw = '0xb45dcee023d74ad1b51ec681a257c13e' + // PID
        '0000000000000000' + // NONCE
        'c6d0c7282624429b8e0dd9d19b6592fa' + // asset
        '000301e240000e00034254430007426974636f696e' + // amount + extra
        '16f90cc73f2b75f0' + // timestamp
        '0001fcb874914fa04c2fb387262b63cbc1120001'; // members
      const [signature, _] = mcl.sign(raw, SIGNER);
      let sig_ser = mcl.g1ToUnifiedHex(signature);
      raw = raw + '0040' + sig_ser.substr(2);
      await expect(registry.mixin(raw)).to.be.revertedWith("invalid signature");
    });

    it("Should succeed", async function () {
      let raw = '0xb45dcee023d74ad1b51ec681a257c13e' + // PID
        '0000000000000000' + // NONCE
        'c6d0c7282624429b8e0dd9d19b6592fa' + // asset
        '000301e240000e00034254430007426974636f696e' + // amount + extra
        '16f90cc73f2b75f0' + // timestamp
        '0001fcb874914fa04c2fb387262b63cbc1120001' + // members
        '0000';
      const [signature, _] = mcl.sign(raw, SIGNER);
      let sig_ser = mcl.g1ToUnifiedHex(signature);
      raw = raw + sig_ser.substr(2);
      let tx = await registry.mixin(raw);
      expect(tx.from).equal(await addr1.getAddress());
      expect(await registry.INBOUND()).to.equal(1);
    });
  });

  describe("Test halt function", function () {
    it("Should fail with invalid signature", async function () {
      let raw = '0x48414c54';
      const [signature, _] = mcl.sign(raw, SIGNER);
      let hraw = mcl.g1ToUnifiedHex(signature);
      await expect(registry.halt(hraw)).to.be.revertedWith("invalid signature");
    });

    it("Should halt successfully", async function () {
      let raw = "0x48414c540000000000000000";
      const [signature, _] = mcl.sign(raw, SIGNER);
      let hraw = mcl.g1ToUnifiedHex(signature);
      let tx = await registry.halt(hraw);
      expect(tx.from).equal(await addr1.getAddress());
    });

    it("Should fail with invalid state", async function () {
      let raw = "0x48414c540000000000000000";
      let [signature, _] = mcl.sign(raw, SIGNER);
      let hraw = mcl.g1ToUnifiedHex(signature);
      await registry.halt(hraw);

      raw = '0xb45dcee023d74ad1b51ec681a257c13e' + // PID
        '0000000000000000' + // NONCE
        'c6d0c7282624429b8e0dd9d19b6592fa' + // asset
        '000301e240000e00034254430007426974636f696e' + // amount + extra
        '16f90cc73f2b75f0' + // timestamp
        '0001fcb874914fa04c2fb387262b63cbc1120001' + // members
        '0000';
      [signature, _] = mcl.sign(raw, SIGNER);
      let sig_ser = mcl.g1ToUnifiedHex(signature);
      let input = raw + sig_ser.substr(2);
      await expect(registry.mixin(input)).to.be.revertedWith("invalid state");
    });

    it("Should not halt without correct NONCE", async function () {
      let raw = '0xb45dcee023d74ad1b51ec681a257c13e' + // PID
        '0000000000000000' + // NONCE
        'c6d0c7282624429b8e0dd9d19b6592fa' + // asset
        '000301e240000e00034254430007426974636f696e' + // amount + extra
        '16f90cc73f2b75f0' + // timestamp
        '0001fcb874914fa04c2fb387262b63cbc1120001' + // members
        '0000';
      let [signature, _] = mcl.sign(raw, SIGNER);
      let sig_ser = mcl.g1ToUnifiedHex(signature);
      let input = raw + sig_ser.substr(2);
      let tx = await registry.mixin(input);
      expect(tx.from).equal(await addr1.getAddress());
      expect(await registry.INBOUND()).to.equal(1);

      [signature, _] = mcl.sign("0x48414c540000000000000000", SIGNER);
      let hraw = mcl.g1ToUnifiedHex(signature);
      await expect(registry.halt(hraw)).to.be.revertedWith("invalid signature");

      [signature, _] = mcl.sign("0x48414c540000000000000001", SIGNER);
      hraw = mcl.g1ToUnifiedHex(signature);
      tx = await registry.halt(hraw);
      expect(tx.from).equal(await addr1.getAddress());
    });

    it("Should succeed after halt toggle", async function () {
      let [signature, _] = mcl.sign("0x48414c540000000000000000", SIGNER);
      let hraw = mcl.g1ToUnifiedHex(signature);
      await registry.halt(hraw);

      let raw = '0xb45dcee023d74ad1b51ec681a257c13e' + // PID
        '0000000000000000' + // NONCE
        'c6d0c7282624429b8e0dd9d19b6592fa' + // asset
        '000301e240000e00034254430007426974636f696e' + // amount + extra
        '16f90cc73f2b75f0' + // timestamp
        '0001fcb874914fa04c2fb387262b63cbc1120001' + // members
        '0000';
      [signature, _] = mcl.sign(raw, SIGNER);
      let sig_ser = mcl.g1ToUnifiedHex(signature);
      let input = raw + sig_ser.substr(2);
      await expect(registry.mixin(input)).to.be.revertedWith("invalid state");

      [signature, _] = mcl.sign("0x48414c540000000000000000", SIGNER);
      hraw = mcl.g1ToUnifiedHex(signature);
      await registry.halt(hraw);

      [signature, _] = mcl.sign(raw, SIGNER);
      sig_ser = mcl.g1ToUnifiedHex(signature);
      input = raw + sig_ser.substr(2);
      let tx = await registry.mixin(input);
      expect(tx.from).equal(await addr1.getAddress());
      expect(await registry.INBOUND()).to.equal(1);
    });
  });

  describe("Test iterate function", function () {
    it("Should fail with invalid state", async function () {
      let {pubkey, secret} = mcl.newKeyPair();
      let pubkey_ser = mcl.g2ToUnifiedHex(pubkey);
      let [sig1, m1] = mcl.sign(pubkey_ser, SIGNER);
      let [sig2, m2] = mcl.sign(mcl.g2ToUnifiedHex(GROUP), secret);
      let input = pubkey_ser + mcl.g1ToUnifiedHex(sig1).substr(2);
      input = input + mcl.g1ToUnifiedHex(sig2).substr(2);
      await expect(registry.iterate(input)).to.be.revertedWith("invalid state");
    });

    it("Should fail with invalid signature", async function () {
      let [signature, _] = mcl.sign("0x48414c540000000000000000", SIGNER);
      let hraw = mcl.g1ToUnifiedHex(signature);
      await registry.halt(hraw);

      let {pubkey, secret} = mcl.newKeyPair();
      let pubkey_ser = mcl.g2ToUnifiedHex(pubkey);
      let [sig1, m1] = mcl.sign(pubkey_ser, SIGNER);
      let [sig2, m2] = mcl.sign(mcl.g2ToUnifiedHex(GROUP), secret);
      let input = pubkey_ser + mcl.g1ToUnifiedHex(sig1).substr(2);
      input = input + mcl.g1ToUnifiedHex(sig2).substr(2);
      await expect(registry.iterate(input)).to.be.revertedWith("invalid signature");
    });

    it("Should iterate to the new group", async function () {
      let [signature, _] = mcl.sign("0x48414c540000000000000000", SIGNER);
      let hraw = mcl.g1ToUnifiedHex(signature);
      await registry.halt(hraw);

      let {pubkey, secret} = mcl.newKeyPair();
      let pubkey_ser = mcl.g2ToUnifiedHex(pubkey);
      let [sig1, m1] = mcl.sign(pubkey_ser, SIGNER);
      let [sig2, m2] = mcl.sign(pubkey_ser, secret);
      let input = pubkey_ser + mcl.g1ToUnifiedHex(sig1).substr(2);
      input = input + mcl.g1ToUnifiedHex(sig2).substr(2);
      let tx = await registry.iterate(input);
      expect(tx.from).equal(await addr1.getAddress());

      for (let i = 0; i < 4; i++) {
        let sge = await registry.GROUP(i);
        let tge = mcl.g2ToBN(pubkey)[i];
        expect(sge.toString()).to.equal(tge.toString());
      }
    });
  });

});
