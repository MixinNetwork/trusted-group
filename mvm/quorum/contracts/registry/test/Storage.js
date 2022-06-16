// We import Chai to use its asserting functions here.
const { expect } = require("chai");

// `describe` is a Mocha function that allows you to organize your tests. It's
// not actually needed, but having your tests organized makes debugging them
// easier. All Mocha functions are available in the global scope.

// `describe` receives the name of a section of your test suite, and a callback.
// The callback must define the tests of that section. This callback can't be
// an async function.
describe("Storage contract", function () {
  // Mocha has four functions that let you hook into the test runner's
  // lifecyle. These are: `before`, `beforeEach`, `after`, `afterEach`.

  // They're very useful to setup the environment for tests, and to clean it
  // up after they run.

  // A common pattern is to declare some variables, and assign them in the
  // `before` and `beforeEach` callbacks.

  let Storage;
  let storage;
  let addr1;
  let addr2;
  let addrs;

  // `beforeEach` will run before each test, re-deploying the contract every
  // time. It receives a callback, which can be async.
  beforeEach(async function () {
    // Get the ContractFactory and Signers here.
    Storage = await ethers.getContractFactory("Storage");
    [addr1, addr2, ...addrs] = await ethers.getSigners();

    // To deploy our contract, we just have to call Token.deploy() and await
    // for it to be deployed(), which happens once its transaction has been
    // mined.
    storage = await Storage.connect(addr2).deploy();
  });

  describe("Transactions", function () {
    it("Should write value", async function () {
      const raw = "0x1234567890";
      const key = ethers.utils.keccak256(raw);
      await storage.write(key, raw);
      expect(await storage.read(key)).to.equal(raw);
      expect(await storage.connect(addr1).read(key)).to.equal(raw);
      expect(await storage.connect(addr2).read(key)).to.equal(raw);
    });

    it("Should fail key check", async function () {
      const raw = "0x1234567890";
      const key = ethers.utils.keccak256(raw);
      await expect(storage.write(raw, raw)).to.be.revertedWith('invalid key or raw');
      expect(await storage.read(key)).to.equal("0x");
      expect(await storage.connect(addr1).read(key)).to.equal("0x");
    });
  });
});
