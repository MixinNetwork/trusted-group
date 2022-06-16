// We import Chai to use its asserting functions here.
import { ethers } from "hardhat";
import { expect, assert } from "chai";
import BN from "bn.js";
import { randomHex } from "./bls/utils";
const mcl = require("./bls/mcl");
import { Registry } from "../typechain-types";
import { TestBLS } from "../typechain-types/test";

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
  let sbls: TestBLS;
  let addr1;
  let addr2;
  let addrs;

  // `beforeEach` will run before each test, re-deploying the contract every
  // time. It receives a callback, which can be async.
  beforeEach(async function () {
    // Get the ContractFactory and Signers here.
    let BLS = await ethers.getContractFactory("TestBLS");
    sbls = await BLS.deploy();

    [addr1, addr2, ...addrs] = await ethers.getSigners();
    let Registry = await ethers.getContractFactory("Registry", {
      //libraries: {
        //BLS: sbls.address,
      //},
    });
    // To deploy our contract, we just have to call Token.deploy() and await
    // for it to be deployed(), which happens once its transaction has been
    // mined.
    registry = await Registry.connect(addr1).deploy("0x1b0b73f760f5a1fc2d3b14b18a1fb5f7d8e93366ac283423c7b6413dd869bf1300a4398f2222cca7c31cd56b4557249bf9f3c538b30b06e3a9a4c830a9b48feb25fc87924ed7906607d59b6e9555230e73b46378252923ac719a328c7235cb03030e7999862a645112eefee572b4f930a8c66b4141d0f8d76558364ed539c03e", "0xb45dcee023d74ad1b51ec681a257c13e");
  });

  describe("Deployment", function () {
    it("Should has the same address", async function () {
      expect(registry.address).to.equal("0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0");
    });
  });
});
